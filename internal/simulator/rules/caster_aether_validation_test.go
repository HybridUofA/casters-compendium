package rules

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateGenerateCasterAetherAllowsControlledCasterWithoutMutation(t *testing.T) {
	state, catalog := validCasterAetherValidationState()
	before := cloneCasterAetherValidationState(state)

	element, amount, err := ValidateGenerateCasterAether(
		&state,
		catalog,
		"player-two",
		"p2-controlled-caster",
	)
	if err != nil {
		t.Fatalf("ValidateGenerateCasterAether() error = %v; want nil", err)
	}
	if element != model.ElementAqua || amount != 2 {
		t.Fatalf("ValidateGenerateCasterAether() = %q/%d; want Aqua/2", element, amount)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateGenerateCasterAetherRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		catalog     func(definitionCatalogForTest) CardCatalog
		playerID    model.PlayerID
		cardID      model.MatchCardID
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{name: "match not in progress", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			state.MatchStatus = model.StatusSetup
		}, wantErrPart: "must be \"In Progress\""},
		{name: "unknown player", playerID: "missing-player", cardID: "p2-controlled-caster", wantErrPart: "not found in players"},
		{name: "card outside acting player's Caster Zone", playerID: "player-two", cardID: "p1-caster", wantErrPart: "not found in caster zone"},
		{name: "missing instance", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			delete(state.CardInstances, "p2-controlled-caster")
		}, wantErrPart: "caster \"p2-controlled-caster\" not found"},
		{name: "Caster Token", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-controlled-caster"]
			instance.CardID = model.CasterTokenCardID
			state.CardInstances["p2-controlled-caster"] = instance
		}, wantErrPart: "is a caster token"},
		{name: "non-printed instance", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-controlled-caster"]
			instance.CardCategory = model.CategoryTokenCard
			state.CardInstances["p2-controlled-caster"] = instance
		}, wantErrPart: "not a printed card"},
		{name: "controlled by other player", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-controlled-caster"]
			instance.Controller = "player-one"
			state.CardInstances["p2-controlled-caster"] = instance
		}, wantErrPart: "controller"},
		{name: "Rested Caster", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-controlled-caster"]
			instance.Orientation = model.OrientationRested
			state.CardInstances["p2-controlled-caster"] = instance
		}, wantErrPart: "must be recovered"},
		{name: "face-down Caster", playerID: "player-two", cardID: "p2-controlled-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-controlled-caster"]
			instance.Face = model.CardFaceDown
			state.CardInstances["p2-controlled-caster"] = instance
		}, wantErrPart: "must be face-up"},
		{name: "nil catalog", catalog: func(definitionCatalogForTest) CardCatalog { return nil }, playerID: "player-two", cardID: "p2-controlled-caster", wantErrPart: "catalog cannot be nil"},
		{name: "missing definition", catalog: func(definitionCatalogForTest) CardCatalog { return definitionCatalogForTest{} }, playerID: "player-two", cardID: "p2-controlled-caster", wantErrPart: "error looking up card"},
		{name: "definition is not a Caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["aqua-caster"]
			card.Type = "Servant"
			catalog["aqua-caster"] = card
			return catalog
		}, playerID: "player-two", cardID: "p2-controlled-caster", wantErrPart: "not a caster"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, defaultCatalog := validCasterAetherValidationState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			catalog := CardCatalog(defaultCatalog)
			if testCase.catalog != nil {
				catalog = testCase.catalog(defaultCatalog)
			}
			before := cloneCasterAetherValidationState(state)

			element, amount, err := ValidateGenerateCasterAether(
				&state,
				catalog,
				testCase.playerID,
				testCase.cardID,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateGenerateCasterAether() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if element != "" || amount != 0 {
				t.Fatalf("rejected validation returned %q/%d; want zero values", element, amount)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateGenerateCasterAetherRejectsNilState(t *testing.T) {
	element, amount, err := ValidateGenerateCasterAether(nil, definitionCatalogForTest{}, "player-one", "caster")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateGenerateCasterAether(nil) error = %v; want nil-state error", err)
	}
	if element != "" || amount != 0 {
		t.Fatalf("ValidateGenerateCasterAether(nil) = %q/%d; want zero values", element, amount)
	}
}

func validCasterAetherValidationState() (model.MatchState, definitionCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-caster": {
				CardID:       "aqua-caster",
				MatchID:      "p1-caster",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
				Face:         model.CardFaceUp,
				Orientation:  model.OrientationRecovered,
			},
			"p2-controlled-caster": {
				CardID:       "aqua-caster",
				MatchID:      "p2-controlled-caster",
				Owner:        "player-one",
				Controller:   "player-two",
				CardCategory: model.CategoryPrintedCard,
				Face:         model.CardFaceUp,
				Orientation:  model.OrientationRecovered,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"p1-caster"}},
			{ID: "player-two", CasterZone: []model.MatchCardID{"p2-controlled-caster"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn:        model.TurnState{Number: 3, ActivePlayer: "player-one", Phase: model.PhaseBattle},
	}
	catalog := definitionCatalogForTest{
		"aqua-caster": gamecards.Card{
			ID:        "aqua-caster",
			Type:      "Caster",
			Element:   "Aqua",
			CostLevel: "2",
		},
	}
	return state, catalog
}

func cloneCasterAetherValidationState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].Exile = slices.Clone(state.Players[index].Exile)
	}
	return clone
}
