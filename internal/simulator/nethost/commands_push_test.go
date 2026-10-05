package nethost

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

func TestMutatingCommandReplyNotBlockedByStalledPeer(t *testing.T) {
	lobby := NewLobby()
	host := NewHost(lobby)

	code, _, _, err := lobby.CreateRoom("Host", "", validTestDeck("Host Deck"), "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if _, err := lobby.JoinRoom(code, "Guest", validTestDeck("Guest Deck"), ""); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusInProgress,
		Turn:        model.TurnState{ActivePlayer: "player-one"},
	}
	match, err := session.NewLocalMatch(state, engine.MatchSeed{First: 1, Second: 2}, lobbyTestCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	if err := lobby.AttachMatch(code, match); err != nil {
		t.Fatalf("AttachMatch() error = %v", err)
	}

	peerWriteStarted := make(chan struct{})
	peerReleased := make(chan struct{})
	var peerWriteOnce sync.Once
	peer := &wsClient{
		roomCode: code,
		playerID: "player-two",
		writeFn: func(ctx context.Context, data []byte) error {
			peerWriteOnce.Do(func() { close(peerWriteStarted) })
			select {
			case <-peerReleased:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
	host.register(peer)
	t.Cleanup(func() {
		close(peerReleased)
		host.unregister(peer)
	})

	req, err := protocol.Encode(protocol.KindCommand, protocol.CommandPayload{
		PlayerID: "player-one",
		Revision: 0,
		Name:     "shuffle_deck",
	})
	if err != nil {
		t.Fatalf("encode command: %v", err)
	}

	start := time.Now()
	raw, err := host.HandleMessage(req)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("HandleMessage(shuffle_deck) error = %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("mutating command blocked for %v; actor reply must not wait on peer push", elapsed)
	}

	env, err := protocol.Decode(raw)
	if err != nil {
		t.Fatalf("decode reply: %v", err)
	}
	if env.Kind != protocol.KindView {
		var errPayload protocol.ErrorPayload
		_ = json.Unmarshal(env.Payload, &errPayload)
		t.Fatalf("kind = %q error=%#v; want view", env.Kind, errPayload)
	}

	select {
	case <-peerWriteStarted:
	case <-time.After(peerPushTimeout + time.Second):
		t.Fatal("peer push was never attempted")
	}
}
