package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestOfferAndResolvePendingChoice(t *testing.T) {
	state := model.MatchState{
		Revision: 3,
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
	}
	if err := OfferPendingChoice(&state, model.PendingChoice{
		PlayerID:     "player-one",
		Kind:         "test_prompt",
		Prompt:       "Pick a mode.",
		AllowDecline: true,
		Options: []model.PendingChoiceOption{
			{ID: "a", Label: "Mode A"},
			{ID: "b", Label: "Mode B"},
		},
		SelectableCardIDs: []model.MatchCardID{"card-1", "card-2"},
	}); err != nil {
		t.Fatalf("OfferPendingChoice() error = %v", err)
	}
	if state.PendingChoice.PlayerID != "player-one" || state.PendingChoice.Kind != "test_prompt" {
		t.Fatalf("PendingChoice = %#v", state.PendingChoice)
	}

	err := ResolvePendingChoice(&state, nil, "player-one", "a", []model.MatchCardID{"card-1"}, 3)
	if err != nil {
		t.Fatalf("ResolvePendingChoice() error = %v", err)
	}
	if state.PendingChoice.PlayerID != "" {
		t.Fatalf("PendingChoice = %#v; want cleared", state.PendingChoice)
	}
	if state.Revision != 4 {
		t.Fatalf("Revision = %d; want 4", state.Revision)
	}
}

func TestDeclinePendingChoiceRequiresAllowDecline(t *testing.T) {
	state := model.MatchState{
		Revision: 1,
		PendingChoice: model.PendingChoice{
			PlayerID: "player-one",
			Kind:     "mandatory",
			Prompt:   "Must choose.",
		},
	}
	err := DeclinePendingChoice(&state, "player-one", 1)
	if err == nil || !strings.Contains(err.Error(), "cannot be declined") {
		t.Fatalf("DeclinePendingChoice() error = %v; want cannot be declined", err)
	}

	state.PendingChoice.AllowDecline = true
	if err := DeclinePendingChoice(&state, "player-one", 1); err != nil {
		t.Fatalf("DeclinePendingChoice() error = %v", err)
	}
	if state.PendingChoice.PlayerID != "" {
		t.Fatalf("PendingChoice = %#v; want cleared", state.PendingChoice)
	}
}

func TestResolvePendingChoiceRejectsInvalidOption(t *testing.T) {
	state := model.MatchState{
		Revision: 2,
		PendingChoice: model.PendingChoice{
			PlayerID: "player-one",
			Options: []model.PendingChoiceOption{
				{ID: "a", Label: "A"},
			},
		},
	}
	err := ResolvePendingChoice(&state, nil, "player-one", "missing", nil, 2)
	if err == nil || !strings.Contains(err.Error(), "not part of the pending choice") {
		t.Fatalf("ResolvePendingChoice() error = %v; want invalid option", err)
	}
}
