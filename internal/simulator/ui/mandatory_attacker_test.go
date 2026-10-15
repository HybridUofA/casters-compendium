package ui

import (
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestViewerHasMandatoryAttackerIgnoresBarriers(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn = model.TurnState{
		Number:       2,
		ActivePlayer: match.ViewerID,
		Phase:        model.PhaseBattle,
	}
	match.Attack.Step = model.BattleStepIdle
	match.Players[0].ServantZone = []simulatorview.CardView{
		{
			MatchID:     "barrier-1",
			CardID:      "barrier-def",
			Orientation: model.OrientationRecovered,
			ShowFace:    true,
		},
	}
	match.Players[1].ServantZone = nil
	definitions := cardLookup{
		"barrier-def": cards.Card{ID: "barrier-def", Name: "Shield Wall", Type: "Barrier"},
	}
	if viewerHasMandatoryAttacker(match, definitions) {
		t.Fatal("recovered Barrier alone must not block ending Battle")
	}

	match.Players[0].ServantZone = append(match.Players[0].ServantZone, simulatorview.CardView{
		MatchID:     "servant-1",
		CardID:      "servant-def",
		Orientation: model.OrientationRecovered,
		ShowFace:    true,
	})
	definitions["servant-def"] = cards.Card{ID: "servant-def", Name: "Crispy Cookie", Type: "Servant"}
	if !viewerHasMandatoryAttacker(match, definitions) {
		t.Fatal("recovered Servant must still require an attack")
	}

	match.Players[0].ServantZone = []simulatorview.CardView{{
		MatchID:     "reversed-1",
		CardID:      "servant-def",
		Orientation: model.OrientationReversed,
		ShowFace:    true,
	}}
	if viewerHasMandatoryAttacker(match, definitions) {
		t.Fatal("reversed Servant must not count as an eligible attacker")
	}
}
