package nethost

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

const maxRoomNameRunes = 48

func normalizeRoomName(roomName, playerName string) string {
	name := strings.Join(strings.Fields(strings.TrimSpace(roomName)), " ")
	if name == "" {
		if playerName == "" {
			return "Open Room"
		}
		return playerName + "'s Room"
	}
	if utf8.RuneCountInString(name) > maxRoomNameRunes {
		runes := []rune(name)
		name = string(runes[:maxRoomNameRunes])
	}
	return name
}

type Room struct {
	Code           string
	Name           string
	PasswordHash   string
	Players        map[string]string
	Spectators     map[string]string
	Decks          map[string]decks.Deck
	Match          *session.LocalMatch
	Sessions       map[string]*session.PlayerSession
	nextSpectatorN int
}

type Lobby struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

func NewLobby() *Lobby {
	return &Lobby{rooms: make(map[string]*Room)}
}

func randomRoomCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	code := make([]byte, 4)
	for i := range code {
		code[i] = letters[rand.IntN(len(letters))]
	}
	return string(code)
}

func hashRoomPassword(password string) string {
	if password == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func (room *Room) passwordMatches(password string) bool {
	if room == nil {
		return false
	}
	expected := room.PasswordHash
	if expected == "" {
		return true
	}
	got := hashRoomPassword(password)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1
}

func (room *Room) IsSpectator(id string) bool {
	if room == nil || room.Spectators == nil {
		return false
	}
	_, ok := room.Spectators[id]
	return ok
}

func (room *Room) DisplayNames() map[string]string {
	if room == nil {
		return nil
	}
	names := make(map[string]string, len(room.Players)+len(room.Spectators))
	for id, name := range room.Players {
		names[id] = name
	}
	for id, name := range room.Spectators {
		names[id] = name
	}
	return names
}

func (lobby *Lobby) CreateRoom(playerName, roomName string, deck decks.Deck, password string) (roomCode, playerID, resolvedName string, err error) {
	if playerName == "" {
		return "", "", "", fmt.Errorf("player name is required")
	}
	if err := deck.Validate(); err != nil {
		return "", "", "", fmt.Errorf("deck: %w", err)
	}
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	for range 16 {
		roomCode = randomRoomCode()
		if _, exists := lobby.rooms[roomCode]; !exists {
			break
		}
		roomCode = ""
	}
	if roomCode == "" {
		return "", "", "", fmt.Errorf("could not allocate room code")
	}
	playerID = "player-one"
	resolvedName = normalizeRoomName(roomName, playerName)
	lobby.rooms[roomCode] = &Room{
		Code:         roomCode,
		Name:         resolvedName,
		PasswordHash: hashRoomPassword(password),
		Players:      map[string]string{playerID: playerName},
		Spectators:   map[string]string{},
		Decks:        map[string]decks.Deck{playerID: deck},
	}
	return roomCode, playerID, resolvedName, nil
}

func (lobby *Lobby) JoinRoom(roomCode, playerName string, deck decks.Deck, password string) (playerID string, err error) {
	if playerName == "" {
		return "", fmt.Errorf("player name is required")
	}
	if err := deck.Validate(); err != nil {
		return "", fmt.Errorf("deck: %w", err)
	}
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	room, ok := lobby.rooms[roomCode]
	if !ok {
		return "", fmt.Errorf("room not found")
	}
	if !room.passwordMatches(password) {
		return "", fmt.Errorf("incorrect password")
	}
	if room.Match != nil {
		return "", fmt.Errorf("match already started")
	}
	if len(room.Players) >= 2 {
		return "", fmt.Errorf("room is full")
	}
	playerID = "player-two"
	room.Players[playerID] = playerName
	if room.Decks == nil {
		room.Decks = make(map[string]decks.Deck)
	}
	room.Decks[playerID] = deck
	return playerID, nil
}

func (lobby *Lobby) JoinSpectator(roomCode, spectatorName, password string) (spectatorID string, err error) {
	if spectatorName == "" {
		return "", fmt.Errorf("player name is required")
	}
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	room, ok := lobby.rooms[roomCode]
	if !ok {
		return "", fmt.Errorf("room not found")
	}
	if !room.passwordMatches(password) {
		return "", fmt.Errorf("incorrect password")
	}
	room.nextSpectatorN++
	spectatorID = fmt.Sprintf("spectator-%d", room.nextSpectatorN)
	if room.Spectators == nil {
		room.Spectators = make(map[string]string)
	}
	room.Spectators[spectatorID] = spectatorName
	if room.Match != nil {
		viewerSession, sessionErr := session.NewPlayerSession(room.Match, model.PlayerID(spectatorID))
		if sessionErr != nil {
			delete(room.Spectators, spectatorID)
			return "", sessionErr
		}
		if room.Sessions == nil {
			room.Sessions = make(map[string]*session.PlayerSession)
		}
		room.Sessions[spectatorID] = viewerSession
	}
	return spectatorID, nil
}

func (lobby *Lobby) ListRooms() []protocol.RoomSummary {
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	out := make([]protocol.RoomSummary, 0, len(lobby.rooms))
	for _, room := range lobby.rooms {
		hostName := room.Players["player-one"]
		openSeats := 2 - len(room.Players)
		if openSeats < 0 {
			openSeats = 0
		}
		out = append(out, protocol.RoomSummary{
			RoomCode:       room.Code,
			RoomName:       room.Name,
			HostName:       hostName,
			PlayerCount:    len(room.Players),
			SpectatorCount: len(room.Spectators),
			HasPassword:    room.PasswordHash != "",
			MatchStarted:   room.Match != nil,
			OpenSeats:      openSeats,
		})
	}
	return out
}

func (lobby *Lobby) AttachMatch(roomCode string, match *session.LocalMatch) error {
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	room, ok := lobby.rooms[roomCode]
	if !ok {
		return fmt.Errorf("room not found")
	}
	if room.Match != nil {
		return fmt.Errorf("match already started")
	}
	if len(room.Players) != 2 {
		return fmt.Errorf("room must contain 2 players")
	}
	if match == nil {
		return fmt.Errorf("match is required")
	}
	playerOne, err := session.NewPlayerSession(match, "player-one")
	if err != nil {
		return err
	}
	playerTwo, err := session.NewPlayerSession(match, "player-two")
	if err != nil {
		return err
	}
	room.Match = match
	room.Sessions = map[string]*session.PlayerSession{
		"player-one": playerOne,
		"player-two": playerTwo,
	}
	for spectatorID := range room.Spectators {
		viewerSession, sessionErr := session.NewPlayerSession(match, model.PlayerID(spectatorID))
		if sessionErr != nil {
			return sessionErr
		}
		room.Sessions[spectatorID] = viewerSession
	}
	return nil
}

func (lobby *Lobby) Session(roomCode, playerID string) (*session.PlayerSession, error) {
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	room, ok := lobby.rooms[roomCode]
	if !ok {
		return nil, fmt.Errorf("room not found")
	}
	playerSession, ok := room.Sessions[playerID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return playerSession, nil
}

func (lobby *Lobby) Room(roomCode string) (*Room, bool) {
	lobby.mu.Lock()
	defer lobby.mu.Unlock()
	room, ok := lobby.rooms[roomCode]
	return room, ok
}
