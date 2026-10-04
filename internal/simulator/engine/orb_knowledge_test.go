package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestPeekOrbMarksKnowledgeForActor(t *testing.T) {
	state := orbKnowledgeStateForTest()
	orbID, err := PeekOrb(&state, "player-one", "player-two", 1, state.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if orbID != "p2-orb-b" {
		t.Fatalf("orbID = %q; want p2-orb-b", orbID)
	}
	if !model.ViewerKnowsCard(&state, "player-one", "p2-orb-b") {
		t.Fatal("actor did not gain knowledge of peeked orb")
	}
	if model.ViewerKnowsCard(&state, "player-two", "p2-orb-b") {
		t.Fatal("peek must not mark knowledge for the orb owner via KnownCards")
	}
	if state.Revision != 2 {
		t.Fatalf("Revision = %d; want 2", state.Revision)
	}
}

func TestRevealOrbMarksKnowledgeForOpponent(t *testing.T) {
	state := orbKnowledgeStateForTest()
	orbID, err := RevealOrb(&state, "player-one", 0, state.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if orbID != "p1-orb-a" {
		t.Fatalf("orbID = %q; want p1-orb-a", orbID)
	}
	if !model.ViewerKnowsCard(&state, "player-two", "p1-orb-a") {
		t.Fatal("opponent did not gain knowledge of revealed orb")
	}
	if model.ViewerKnowsCard(&state, "player-one", "p1-orb-a") {
		t.Fatal("reveal should mark the opponent, not re-mark the owner in KnownCards")
	}
}

func TestPeekOrbRejectsOutOfRange(t *testing.T) {
	state := orbKnowledgeStateForTest()
	before := state.Revision
	_, err := PeekOrb(&state, "player-one", "player-two", 9, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v; want out-of-range", err)
	}
	if state.Revision != before {
		t.Fatal("rejected peek bumped revision")
	}
}

func orbKnowledgeStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-orb-a": {MatchID: "p1-orb-a", CardID: "card-a", Owner: "player-one", Controller: "player-one"},
			"p2-orb-a": {MatchID: "p2-orb-a", CardID: "card-x", Owner: "player-two", Controller: "player-two"},
			"p2-orb-b": {MatchID: "p2-orb-b", CardID: "card-y", Owner: "player-two", Controller: "player-two"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Orbs: []model.MatchCardID{"p1-orb-a"}},
			{ID: "player-two", Orbs: []model.MatchCardID{"p2-orb-a", "p2-orb-b"}},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    1,
	}
}
