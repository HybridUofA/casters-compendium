package nethost

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

type Room struct {
	Code     string
	Players  map[string]string
	Decks    map[string]decks.Deck
	Match    *session.LocalMatch
	Sessions map[string]*session.PlayerSession
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

func (lobby *Lobby) CreateRoom(playerName string, deck decks.Deck) (roomCode string, playerID string, err error) {
	if playerName == "" {
		return "", "", fmt.Errorf("player name is required")
	}
	if err := deck.Validate(); err != nil {
		return "", "", fmt.Errorf("deck: %w", err)
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
		return "", "", fmt.Errorf("could not allocate room code")
	}
	playerID = "player-one"
	lobby.rooms[roomCode] = &Room{
		Code:    roomCode,
		Players: map[string]string{playerID: playerName},
		Decks:   map[string]decks.Deck{playerID: deck},
	}
	return roomCode, playerID, nil
}

func (lobby *Lobby) JoinRoom(roomCode, playerName string, deck decks.Deck) (playerID string, err error) {
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
