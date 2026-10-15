package engine

import (
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type barrierCatalog map[string]gamecards.Card

func (c barrierCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestBarriersAreNotAttackTargetsOrReversedBlockers(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p2-barrier"] = battleCard("p2-barrier", "player-two", model.OrientationRecovered)
	state.Players[1].ServantZone = append(state.Players[1].ServantZone, "p2-barrier")
	catalog := barrierCatalog{
		"definition-p1-attacker": {ID: "definition-p1-attacker", Type: "Servant", Attack: "3000", Defense: "2000"},
		"definition-p2-defender": {ID: "definition-p2-defender", Type: "Servant", Attack: "2000", Defense: "1000"},
		"definition-p2-barrier":  {ID: "definition-p2-barrier", Type: "Barrier", Name: "Wall"},
		"definition-p2-orb-1":    {ID: "definition-p2-orb-1", Type: "Servant"},
		"definition-p2-orb-2":    {ID: "definition-p2-orb-2", Type: "Servant"},
	}

	err := DeclareAttack(
		&state, catalog, "player-one", "p1-attacker",
		model.AttackTargetServant, "p2-barrier", state.Revision,
	)
	if err == nil || !strings.Contains(err.Error(), "barriers cannot be attacked") {
		t.Fatalf("DeclareAttack barrier target error = %v; want barriers cannot be attacked", err)
	}

	// A reversed barrier must not block a player attack.
	barrier := state.CardInstances["p2-barrier"]
	barrier.Orientation = model.OrientationReversed
	state.CardInstances["p2-barrier"] = barrier
	if err := DeclareAttack(
		&state, catalog, "player-one", "p1-attacker",
		model.AttackTargetPlayer, "", state.Revision,
	); err != nil {
		t.Fatalf("player attack with reversed barrier blocked: %v", err)
	}
}

func TestCountServantsExcludesBarriers(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p2-barrier"] = battleCard("p2-barrier", "player-two", model.OrientationRecovered)
	state.Players[1].ServantZone = []model.MatchCardID{"p2-defender", "p2-barrier"}
	catalog := barrierCatalog{
		"definition-p2-defender": {ID: "definition-p2-defender", Type: "Servant"},
		"definition-p2-barrier":  {ID: "definition-p2-barrier", Type: "Barrier"},
	}
	if got := countServants(&state, catalog, 1); got != 1 {
		t.Fatalf("countServants = %d; want 1 (barrier excluded)", got)
	}
}

func TestSetFieldOrientationRejectsReversedBarrier(t *testing.T) {
	state := battleStateForTest()
	state.CardInstances["p2-barrier"] = battleCard("p2-barrier", "player-two", model.OrientationRecovered)
	state.Players[1].ServantZone = []model.MatchCardID{"p2-barrier"}
	catalog := barrierCatalog{
		"definition-p2-barrier": {ID: "definition-p2-barrier", Type: "Barrier"},
	}
	err := SetFieldOrientation(&state, catalog, "player-two", "p2-barrier", model.OrientationReversed, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "barrier orientation") {
		t.Fatalf("SetFieldOrientation error = %v; want barrier orientation rejection", err)
	}
}
