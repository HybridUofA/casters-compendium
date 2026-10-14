package engine

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestClearTurnEventFlagsResetsCountersAndTempOverlays(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"s1": {
				CardID:                   "1",
				MatchID:                  "s1",
				BattledThisTurn:          true,
				GrantedSlowStartUntilEOT: true,
				LosesSlowStartUntilEOT:   true,
			},
		},
		TurnFlags: model.TurnFlags{ServantsEntered: [2]int{2, 1}},
		ContinuousUntilEOT: model.ContinuousUntilEOT{
			PlayerCannotBeAttacked: [2]bool{true, false},
		},
	}
	clearTurnEventFlags(&state)
	if state.TurnFlags.ServantsEntered != [2]int{} {
		t.Fatalf("TurnFlags not cleared: %#v", state.TurnFlags)
	}
	if state.ContinuousUntilEOT.PlayerCannotBeAttacked[0] {
		t.Fatal("ContinuousUntilEOT not cleared")
	}
	got := state.CardInstances["s1"]
	if got.BattledThisTurn || got.GrantedSlowStartUntilEOT || got.LosesSlowStartUntilEOT {
		t.Fatalf("card overlays not cleared: %#v", got)
	}
}

func TestNoteServantEnteredBumpsFlagAndEnteredOnTurn(t *testing.T) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"s1": {CardID: "1", MatchID: "s1", Controller: "player-one"},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		Turn: model.TurnState{Number: 3},
	}
	noteServantEntered(&state, "player-one", "s1")
	if state.TurnFlags.ServantsEntered[0] != 1 {
		t.Fatalf("ServantsEntered = %d; want 1", state.TurnFlags.ServantsEntered[0])
	}
	if state.CardInstances["s1"].EnteredOnTurn != 3 {
		t.Fatalf("EnteredOnTurn = %d; want 3", state.CardInstances["s1"].EnteredOnTurn)
	}
}
