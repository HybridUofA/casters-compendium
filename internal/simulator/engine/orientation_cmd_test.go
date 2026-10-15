package engine

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestSetFieldOrientationCyclesServantPositions(t *testing.T) {
	state := battleStateForTest()
	rev := state.Revision
	if err := SetFieldOrientation(&state, nil, "player-one", "p1-attacker", model.OrientationRested, rev); err != nil {
		t.Fatalf("SetFieldOrientation(Rested) error = %v", err)
	}
	if state.CardInstances["p1-attacker"].Orientation != model.OrientationRested {
		t.Fatalf("orientation = %q; want Rested", state.CardInstances["p1-attacker"].Orientation)
	}
	rev = state.Revision
	if err := SetFieldOrientation(&state, nil, "player-one", "p1-attacker", model.OrientationReversed, rev); err != nil {
		t.Fatalf("SetFieldOrientation(Reversed) error = %v", err)
	}
	if state.CardInstances["p1-attacker"].Orientation != model.OrientationReversed {
		t.Fatalf("orientation = %q; want Reversed", state.CardInstances["p1-attacker"].Orientation)
	}
	rev = state.Revision
	if err := SetFieldOrientation(&state, nil, "player-one", "p1-attacker", model.OrientationRecovered, rev); err != nil {
		t.Fatalf("SetFieldOrientation(Recovered) error = %v", err)
	}
	if state.CardInstances["p1-attacker"].Orientation != model.OrientationRecovered {
		t.Fatalf("orientation = %q; want Recovered", state.CardInstances["p1-attacker"].Orientation)
	}
}

func TestSetFieldOrientationRejectsOpponentCard(t *testing.T) {
	state := battleStateForTest()
	err := SetFieldOrientation(&state, nil, "player-one", "p2-defender", model.OrientationRested, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "controller") {
		t.Fatalf("error = %v; want controller rejection", err)
	}
}
