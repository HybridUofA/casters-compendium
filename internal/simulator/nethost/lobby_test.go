package nethost

import (
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

func TestCreateRoomAssignsHostSeatAndStoresRoom(t *testing.T) {
	lobby := NewLobby()

	code, playerID, _, err := lobby.CreateRoom("Hybrid", "Friday Night", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if len(code) != 4 {
		t.Fatalf("room code %q; want 4 characters", code)
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			t.Fatalf("room code %q contains non A-Z character", code)
		}
	}
	if playerID != "player-one" {
		t.Fatalf("playerID = %q; want player-one", playerID)
	}

	lobby.mu.Lock()
	room := lobby.rooms[code]
	lobby.mu.Unlock()
	if room == nil {
		t.Fatal("created room was not stored")
	}
	if room.Players["player-one"] != "Hybrid" {
		t.Fatalf("Players = %#v", room.Players)
	}
	if room.Name != "Friday Night" {
		t.Fatalf("room.Name = %q; want Friday Night", room.Name)
	}
	rooms := lobby.ListRooms()
	if len(rooms) != 1 || rooms[0].RoomName != "Friday Night" || rooms[0].HostName != "Hybrid" {
		t.Fatalf("ListRooms() = %#v", rooms)
	}
}

func TestNormalizeRoomNameDefaultsAndTruncates(t *testing.T) {
	if got := normalizeRoomName("  ", "Hybrid"); got != "Hybrid's Room" {
		t.Fatalf("default name = %q", got)
	}
	long := strings.Repeat("a", 80)
	if got := normalizeRoomName(long, "Hybrid"); len([]rune(got)) != maxRoomNameRunes {
		t.Fatalf("truncated length = %d; want %d", len([]rune(got)), maxRoomNameRunes)
	}
}

func TestJoinRoomAssignsGuestSeat(t *testing.T) {
	lobby := NewLobby()
	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	playerID, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), "")
	if err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if playerID != "player-two" {
		t.Fatalf("playerID = %q; want player-two", playerID)
	}

	lobby.mu.Lock()
	room := lobby.rooms[code]
	lobby.mu.Unlock()
	if room.Players["player-two"] != "Guest" || len(room.Players) != 2 {
		t.Fatalf("Players = %#v", room.Players)
	}
}

func TestJoinRoomRejectsMissingFullAndBlankNames(t *testing.T) {
	lobby := NewLobby()
	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), ""); err != nil {
		t.Fatalf("first JoinRoom() error = %v", err)
	}

	tests := []struct {
		name       string
		roomCode   string
		playerName string
		wantErr    string
	}{
		{name: "missing room", roomCode: "ZZZZ", playerName: "Other", wantErr: "room not found"},
		{name: "full room", roomCode: code, playerName: "Late", wantErr: "room is full"},
		{name: "blank joiner", roomCode: code, playerName: "", wantErr: "player name is required"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := lobby.JoinRoom(testCase.roomCode, testCase.playerName, validTestDeck("Other Deck"), "")
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("JoinRoom() error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}

	if _, _, _, err := lobby.CreateRoom("", "", validTestDeck("Blank"), ""); err == nil || !strings.Contains(err.Error(), "player name is required") {
		t.Fatalf("CreateRoom(\"\") error = %v; want blank-name error", err)
	}
}

func TestRoomPasswordAndSpectatorLobby(t *testing.T) {
	lobby := NewLobby()
	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "secret")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), "wrong"); err == nil || !strings.Contains(err.Error(), "incorrect password") {
		t.Fatalf("JoinRoom(wrong password) error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), "secret"); err != nil {
		t.Fatalf("JoinRoom(correct password) error = %v", err)
	}

	spectatorID, err := lobby.JoinSpectator(code, "Watcher", "secret")
	if err != nil {
		t.Fatalf("JoinSpectator() error = %v", err)
	}
	if !strings.HasPrefix(spectatorID, "spectator-") {
		t.Fatalf("spectatorID = %q", spectatorID)
	}

	rooms := lobby.ListRooms()
	if len(rooms) != 1 || rooms[0].RoomCode != code || !rooms[0].HasPassword || rooms[0].SpectatorCount != 1 {
		t.Fatalf("ListRooms() = %#v", rooms)
	}

	match := localMatchForTest(t)
	if err := lobby.AttachMatch(code, match); err != nil {
		t.Fatalf("AttachMatch() error = %v", err)
	}
	specSession, err := lobby.Session(code, spectatorID)
	if err != nil {
		t.Fatalf("Session(spectator) error = %v", err)
	}
	view, err := specSession.View()
	if err != nil {
		t.Fatalf("spectator View() error = %v", err)
	}
	if !view.Spectator {
		t.Fatalf("expected spectator view, got %#v", view)
	}
}

func TestAttachMatchStoresSessionsForBothSeats(t *testing.T) {
	lobby := NewLobby()
	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), ""); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	match := localMatchForTest(t)
	if err := lobby.AttachMatch(code, match); err != nil {
		t.Fatalf("AttachMatch() error = %v", err)
	}

	playerOne, err := lobby.Session(code, "player-one")
	if err != nil {
		t.Fatalf("Session(player-one) error = %v", err)
	}
	playerTwo, err := lobby.Session(code, "player-two")
	if err != nil {
		t.Fatalf("Session(player-two) error = %v", err)
	}
	viewOne, err := playerOne.View()
	if err != nil {
		t.Fatalf("playerOne.View() error = %v", err)
	}
	viewTwo, err := playerTwo.View()
	if err != nil {
		t.Fatalf("playerTwo.View() error = %v", err)
	}
	if viewOne.ViewerID != "player-one" || viewTwo.ViewerID != "player-two" {
		t.Fatalf("viewer IDs = %q / %q", viewOne.ViewerID, viewTwo.ViewerID)
	}
}

func TestAttachMatchAndSessionRejectInvalidStates(t *testing.T) {
	lobby := NewLobby()
	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	match := localMatchForTest(t)

	if err := lobby.AttachMatch(code, match); err == nil || !strings.Contains(err.Error(), "room must contain 2 players") {
		t.Fatalf("AttachMatch(one player) error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), ""); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if err := lobby.AttachMatch("ZZZZ", match); err == nil || !strings.Contains(err.Error(), "room not found") {
		t.Fatalf("AttachMatch(missing) error = %v", err)
	}
	if err := lobby.AttachMatch(code, nil); err == nil || !strings.Contains(err.Error(), "match is required") {
		t.Fatalf("AttachMatch(nil) error = %v", err)
	}
	if err := lobby.AttachMatch(code, match); err != nil {
		t.Fatalf("AttachMatch() error = %v", err)
	}
	if err := lobby.AttachMatch(code, match); err == nil || !strings.Contains(err.Error(), "match already started") {
		t.Fatalf("AttachMatch(second) error = %v", err)
	}
	if _, err := lobby.Session(code, "player-three"); err == nil || !strings.Contains(err.Error(), "session not found") {
		t.Fatalf("Session(missing seat) error = %v", err)
	}
	if _, err := lobby.Session("ZZZZ", "player-one"); err == nil || !strings.Contains(err.Error(), "room not found") {
		t.Fatalf("Session(missing room) error = %v", err)
	}
}

type lobbyTestCatalog struct{}

func (lobbyTestCatalog) FindByID(string) (gamecards.Card, bool) {
	return gamecards.Card{}, false
}

func localMatchForTest(t *testing.T) *session.LocalMatch {
	t.Helper()
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusSetup,
		Turn:        model.TurnState{ActivePlayer: "player-one"},
	}
	match, err := session.NewLocalMatch(state, engine.MatchSeed{First: 1, Second: 2}, lobbyTestCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	return match
}

func validTestDeck(name string) decks.Deck {
	return decks.Deck{
		SchemaVersion: 1,
		Name:          name,
	}
}
