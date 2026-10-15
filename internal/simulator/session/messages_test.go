package session

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestSendChatAppearsInBothPlayerViews(t *testing.T) {
	state := sessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	if _, err := playerOne.SendChat("hello from one"); err != nil {
		t.Fatalf("SendChat() error = %v", err)
	}
	playerTwoView, err := playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() error = %v", err)
	}
	if len(playerTwoView.Messages) != 1 {
		t.Fatalf("Messages len = %d; want 1", len(playerTwoView.Messages))
	}
	msg := playerTwoView.Messages[0]
	if msg.Kind != view.MessageKindChat || msg.PlayerID != state.Players[0].ID || msg.Text != "hello from one" {
		t.Fatalf("chat message = %#v", msg)
	}
}

func TestSendChatRejectsEmptyAndOverlong(t *testing.T) {
	state := sessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession() error = %v", err)
	}
	if _, err := playerOne.SendChat("   "); err == nil {
		t.Fatal("empty chat should fail")
	}
	long := strings.Repeat("x", maxChatRunes+1)
	if _, err := playerOne.SendChat(long); err == nil {
		t.Fatal("overlong chat should fail")
	}
}
