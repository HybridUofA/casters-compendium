package engine

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestRecoverPlayerCardsRecoversOnlySpecifiedPlayersRestedFieldCards(t *testing.T) {
	state := recoveryStateForTest()
	beforeOutgoing := state.CardInstances["p1-rested-caster"]
	beforeReversed := state.CardInstances["p2-reversed-servant"]

	if err := recoverPlayerCards(&state, "player-two"); err != nil {
		t.Fatalf("recoverPlayerCards() error = %v; want nil", err)
	}
	for _, cardID := range []model.MatchCardID{"p2-rested-caster", "p2-rested-servant"} {
		if state.CardInstances[cardID].Orientation != model.OrientationRecovered {
			t.Fatalf("card %q orientation = %q; want Recovered", cardID, state.CardInstances[cardID].Orientation)
		}
	}
	if state.CardInstances["p2-reversed-servant"] != beforeReversed {
		t.Fatalf("Recovery changed Reversed card\n before: %#v\n  after: %#v", beforeReversed, state.CardInstances["p2-reversed-servant"])
	}
	if state.CardInstances["p1-rested-caster"] != beforeOutgoing {
		t.Fatal("Recovery changed the outgoing player's Rested Caster")
	}
}

func TestRecoverPlayerCardsRejectsInvalidStateWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		playerID    model.PlayerID
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{
			name:        "unknown player",
			playerID:    "missing-player",
			wantErrPart: "not found in match state",
		},
		{
			name:     "missing Caster instance",
			playerID: "player-two",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p2-rested-caster")
			},
			wantErrPart: "does not exist",
		},
		{
			name:     "missing Servant instance after valid Caster",
			playerID: "player-two",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p2-rested-servant")
			},
			wantErrPart: "does not exist",
		},
		{
			name:     "Caster controlled by other player",
			playerID: "player-two",
			mutate: func(state *model.MatchState) {
				instance := state.CardInstances["p2-rested-caster"]
				instance.Controller = "player-one"
				state.CardInstances["p2-rested-caster"] = instance
			},
			wantErrPart: "does not belong",
		},
		{
			name:     "Servant controlled by other player",
			playerID: "player-two",
			mutate: func(state *model.MatchState) {
				instance := state.CardInstances["p2-rested-servant"]
				instance.Controller = "player-one"
				state.CardInstances["p2-rested-servant"] = instance
			},
			wantErrPart: "does not belong",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := recoveryStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneRecoveryState(state)

			err := recoverPlayerCards(&state, testCase.playerID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("recoverPlayerCards() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected Recovery mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestRecoverPlayerCardsRejectsNilState(t *testing.T) {
	err := recoverPlayerCards(nil, "player-one")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("recoverPlayerCards(nil) error = %v; want nil-state error", err)
	}
}

func TestCompleteEndPhaseRecoversIncomingPlayerAtomically(t *testing.T) {
	state := recoveryStateForTest()
	state.Turn.Phase = model.PhaseEnd
	state.Players[0].Aether = model.AetherPool{Aes: 1}
	state.Players[1].Aether = model.AetherPool{NonElemental: 2}

	if err := CompleteCurrentPhase(&state, "player-one", state.Revision); err != nil {
		t.Fatalf("CompleteCurrentPhase(End) error = %v", err)
	}
	if state.Turn.ActivePlayer != "player-two" ||
		state.Turn.Number != 2 ||
		state.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("turn handoff = %#v; want player-two turn 2 Recovery", state.Turn)
	}
	if state.CardInstances["p2-rested-caster"].Orientation != model.OrientationRecovered ||
		state.CardInstances["p2-rested-servant"].Orientation != model.OrientationRecovered {
		t.Fatal("End completion did not recover the incoming player's Rested cards")
	}
	if state.CardInstances["p1-rested-caster"].Orientation != model.OrientationRested {
		t.Fatal("End completion recovered the outgoing player's card")
	}
	if state.Players[0].Aether != (model.AetherPool{}) ||
		state.Players[1].Aether != (model.AetherPool{}) {
		t.Fatal("End completion did not clear both Aether pools")
	}
	if state.Revision != 1 {
		t.Fatalf("revision = %d; want 1", state.Revision)
	}
}

func TestCompleteEndPhaseRecoveryFailurePreservesCompleteState(t *testing.T) {
	state := recoveryStateForTest()
	state.Turn.Phase = model.PhaseEnd
	state.Players[0].Aether = model.AetherPool{Void: 1}
	state.Players[1].Aether = model.AetherPool{NonElemental: 2}
	delete(state.CardInstances, "p2-rested-servant")
	before := cloneRecoveryState(state)

	err := CompleteCurrentPhase(&state, "player-one", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "error recovering cards") {
		t.Fatalf("CompleteCurrentPhase(End) error = %v; want Recovery error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("failed End Recovery mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func recoveryStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-rested-caster":     recoveryCard("p1-rested-caster", "player-one", model.OrientationRested),
			"p2-rested-caster":     recoveryCard("p2-rested-caster", "player-two", model.OrientationRested),
			"p2-recovered-caster":  recoveryCard("p2-recovered-caster", "player-two", model.OrientationRecovered),
			"p2-rested-servant":    recoveryCard("p2-rested-servant", "player-two", model.OrientationRested),
			"p2-reversed-servant":  recoveryCard("p2-reversed-servant", "player-two", model.OrientationReversed),
			"p2-recovered-servant": recoveryCard("p2-recovered-servant", "player-two", model.OrientationRecovered),
		},
		Players: [2]model.PlayerState{
			{
				ID:         "player-one",
				CasterZone: []model.MatchCardID{"p1-rested-caster"},
			},
			{
				ID:          "player-two",
				CasterZone:  []model.MatchCardID{"p2-rested-caster", "p2-recovered-caster"},
				ServantZone: []model.MatchCardID{"p2-rested-servant", "p2-reversed-servant", "p2-recovered-servant"},
			},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseRecovery,
		},
	}
}

func recoveryCard(
	matchID model.MatchCardID,
	controller model.PlayerID,
	orientation model.CardOrientation,
) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CardID("definition-" + string(matchID)),
		MatchID:      matchID,
		Owner:        controller,
		Controller:   controller,
		CardCategory: model.CategoryPrintedCard,
		Face:         model.CardFaceUp,
		Orientation:  orientation,
	}
}

func cloneRecoveryState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].Deck = slices.Clone(state.Players[index].Deck)
		clone.Players[index].Hand = slices.Clone(state.Players[index].Hand)
		clone.Players[index].Orbs = slices.Clone(state.Players[index].Orbs)
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].ServantZone = slices.Clone(state.Players[index].ServantZone)
		clone.Players[index].Graveyard = slices.Clone(state.Players[index].Graveyard)
	}
	return clone
}
