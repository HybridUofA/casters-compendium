package ai

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestPassAgentChoosesPassWhenHoldingEmptyPriority(t *testing.T) {
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseMain,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{ID: "player-two", OpeningHandFinalized: true},
		},
	}
	action, err := PassAgent{}.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ActionPass {
		t.Fatalf("Choose() kind = %q; want %q", action.Kind, ActionPass)
	}
}

func TestPassAgentKeepsOpeningHand(t *testing.T) {
	match := view.MatchView{
		ViewerID:    "player-two",
		MatchStatus: model.StatusSetup,
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{ID: "player-two", OpeningHandFinalized: false},
		},
	}
	action, err := PassAgent{}.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ActionKeepOpeningHand {
		t.Fatalf("Choose() kind = %q; want %q", action.Kind, ActionKeepOpeningHand)
	}
}
