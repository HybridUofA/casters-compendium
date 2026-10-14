package tsubasa

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/ai"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type stubCatalog map[string]gamecards.Card

func (c stubCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, ok := c[id]
	return card, ok
}

func TestChooseDeclaresAttackWhenAble(t *testing.T) {
	agent := Agent{Catalog: stubCatalog{}}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseBattle,
		},
		Players: [2]view.PlayerView{
			{
				ID:                   "player-one",
				OpeningHandFinalized: true,
				ServantZone: []view.CardView{
					{MatchID: "enemy-s1", Orientation: model.OrientationRecovered},
				},
			},
			{
				ID:                   "player-two",
				OpeningHandFinalized: true,
				ServantZone: []view.CardView{
					{MatchID: "ally-s1", Orientation: model.OrientationRecovered},
				},
			},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionDeclareAttack {
		t.Fatalf("Choose() kind = %q; want %q", action.Kind, ai.ActionDeclareAttack)
	}
	if action.CardID != "ally-s1" {
		t.Fatalf("attacker = %q; want ally-s1", action.CardID)
	}
	if action.TargetKind != model.AttackTargetPlayer {
		t.Fatalf("target kind = %q; want player (no reverse blockers)", action.TargetKind)
	}
}

func TestChoosePrefersPassionWingRestBeforeAttack(t *testing.T) {
	catalog := stubCatalog{
		"pw": {ID: "pw", Name: "Passion Wing Tsubasa", Type: "Caster", CostLevel: "3"},
	}
	agent := Agent{Catalog: catalog}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseBattle,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{
				ID:                   "player-two",
				OpeningHandFinalized: true,
				CasterZone: []view.CardView{
					{
						MatchID:     "pw-field",
						CardID:      "pw",
						Orientation: model.OrientationRecovered,
						Face:        model.CardFaceUp,
						ShowFace:    true,
					},
				},
				ServantZone: []view.CardView{
					{MatchID: "ally-s1", Orientation: model.OrientationRecovered},
				},
			},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionActivateRest {
		t.Fatalf("Choose() kind = %q; want Passion Wing Rest first", action.Kind)
	}
	if action.CardID != "pw-field" {
		t.Fatalf("Rest card = %q; want pw-field", action.CardID)
	}
}

func TestChoosePassesWhenNothingLegal(t *testing.T) {
	agent := Agent{Catalog: stubCatalog{}}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseCall,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{ID: "player-two", OpeningHandFinalized: true},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionCompletePhase {
		t.Fatalf("Choose() kind = %q; want %q", action.Kind, ai.ActionCompletePhase)
	}
}

func TestChooseCallsPassionWingFaceUp(t *testing.T) {
	catalog := stubCatalog{
		"pw1": {ID: "pw1", Name: "Passion Wing", Type: "Caster", CostLevel: "1", Element: "Ignus"},
		"other": {ID: "other", Name: "Other Caster", Type: "Caster", CostLevel: "1", Element: "Ignus"},
	}
	agent := Agent{Catalog: catalog}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseCall,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{
				ID:                   "player-two",
				OpeningHandFinalized: true,
				Hand: []view.CardView{
					{MatchID: "hand-other", CardID: "other"},
					{MatchID: "hand-pw", CardID: "pw1"},
				},
			},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionCallFaceUpLv1 {
		t.Fatalf("Choose() kind = %q; want face-up Call (not face-down)", action.Kind)
	}
	if action.CardID != "hand-pw" {
		t.Fatalf("Call card = %q; want Passion Wing hand-pw", action.CardID)
	}
}

func TestChooseProducesAetherWhenCannotAffordCast(t *testing.T) {
	catalog := stubCatalog{
		"pw":  {ID: "pw", Name: "Passion Wing", Type: "Caster", CostLevel: "1", Element: "Ignus"},
		"srv": {ID: "srv", Name: "Ignus Buddy", Type: "Servant", CostLevel: "2", Element: "Ignus"},
	}
	agent := Agent{Catalog: catalog}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseMain,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{
				ID:                   "player-two",
				OpeningHandFinalized: true,
				Aether:               model.AetherPool{},
				Hand: []view.CardView{
					{MatchID: "hand-srv", CardID: "srv"},
				},
				CasterZone: []view.CardView{
					{
						MatchID:     "pw-field",
						CardID:      "pw",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
			},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionGenerateCasterAether {
		t.Fatalf("Choose() kind = %q; want generate aether before casting", action.Kind)
	}
	if action.CardID != "pw-field" {
		t.Fatalf("aether source = %q; want pw-field", action.CardID)
	}
}

func TestChooseCastsWhenAetherAvailable(t *testing.T) {
	catalog := stubCatalog{
		"srv": {ID: "srv", Name: "Ignus Buddy", Type: "Servant", CostLevel: "2", Element: "Ignus"},
	}
	agent := Agent{Catalog: catalog}
	match := view.MatchView{
		ViewerID:             "player-two",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-two",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseMain,
		},
		Players: [2]view.PlayerView{
			{ID: "player-one", OpeningHandFinalized: true},
			{
				ID:                   "player-two",
				OpeningHandFinalized: true,
				Aether:               model.AetherPool{Ignus: 1, NonElemental: 1},
				Hand: []view.CardView{
					{MatchID: "hand-srv", CardID: "srv"},
				},
			},
		},
	}
	action, err := agent.Choose(match)
	if err != nil {
		t.Fatalf("Choose() error = %v", err)
	}
	if action.Kind != ai.ActionCastServant {
		t.Fatalf("Choose() kind = %q; want cast servant", action.Kind)
	}
	if action.CardID != "hand-srv" {
		t.Fatalf("cast card = %q; want hand-srv", action.CardID)
	}
}
