package rules

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateCastServantAcceptsEveryElementWithoutMutation(t *testing.T) {
	tests := []struct {
		element string
		payment model.AetherPayment
	}{
		{element: "Aes", payment: model.AetherPayment{Aes: 1, NonElemental: 1}},
		{element: " aqua ", payment: model.AetherPayment{Aqua: 1, NonElemental: 1}},
		{element: "IGNUS", payment: model.AetherPayment{Ignus: 1, NonElemental: 1}},
		{element: "Luna", payment: model.AetherPayment{Luna: 1, NonElemental: 1}},
		{element: "Silva", payment: model.AetherPayment{Silva: 1, NonElemental: 1}},
		{element: "Solis", payment: model.AetherPayment{Solis: 1, NonElemental: 1}},
		{element: "Terra", payment: model.AetherPayment{Terra: 1, NonElemental: 1}},
		{element: "Void", payment: model.AetherPayment{Void: 1, NonElemental: 1}},
	}

	for _, testCase := range tests {
		t.Run(strings.TrimSpace(testCase.element), func(t *testing.T) {
			state, catalog := servantCastStateForTest()
			definition := catalog["printed-servant"]
			definition.Element = testCase.element
			catalog[definition.ID] = definition
			before := state

			err := ValidateCastServant(
				&state,
				catalog,
				"player-one",
				"servant-one",
				testCase.payment,
				model.OrientationRecovered,
			)
			if err != nil {
				t.Fatalf("ValidateCastServant() error = %v", err)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("ValidateCastServant() mutated state:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func TestValidateCastServantAcceptsZeroCost(t *testing.T) {
	state, catalog := servantCastStateForTest()
	definition := catalog["printed-servant"]
	definition.CostLevel = "0"
	catalog[definition.ID] = definition

	if err := ValidateCastServant(
		&state,
		catalog,
		"player-one",
		"servant-one",
		model.AetherPayment{},
		model.OrientationRecovered,
	); err != nil {
		t.Fatalf("ValidateCastServant() rejected zero-cost Servant: %v", err)
	}
}

func TestValidateCastServantRejectsInvalidCast(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*model.MatchState, definitionCatalogForTest)
		payment model.AetherPayment
		wantErr string
	}{
		{
			name: "acting player is not active player",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				state.Turn.ActivePlayer = "player-two"
			},
			wantErr: "active player",
		},
		{
			name: "wrong phase",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				state.Turn.Phase = model.PhaseBattle
			},
			wantErr: "phase must be Main",
		},
		{
			name: "nonempty chase",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				state.ChaseLinks = model.Chase{{ID: 1}}
			},
			wantErr: "chase must be empty",
		},
		{
			name: "card not in hand",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				state.Players[0].Hand = nil
			},
			wantErr: "was not found in the acting player's hand",
		},
		{
			name: "card owned by opponent",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				instance := state.CardInstances["servant-one"]
				instance.Owner = "player-two"
				state.CardInstances["servant-one"] = instance
			},
			wantErr: "must be owned by acting player",
		},
		{
			name: "token card",
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				instance := state.CardInstances["servant-one"]
				instance.CardCategory = model.CategoryTokenCard
				state.CardInstances["servant-one"] = instance
			},
			wantErr: "must be a printed card",
		},
		{
			name: "not a Servant",
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.Type = "Conjure"
				catalog[definition.ID] = definition
			},
			wantErr: "must be a servant",
		},
		{
			name: "malformed cost",
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.CostLevel = "many"
				catalog[definition.ID] = definition
			},
			wantErr: "not a valid integer",
		},
		{
			name: "negative cost",
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.CostLevel = "-1"
				catalog[definition.ID] = definition
			},
			wantErr: "cannot be negative",
		},
		{
			name: "unsupported element",
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.Element = ""
				catalog[definition.ID] = definition
			},
			wantErr: "blank or unsupported",
		},
		{
			name:    "invalid payment",
			payment: model.AetherPayment{Aes: 1},
			wantErr: "error paying for servant",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, catalog := servantCastStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state, catalog)
			}
			payment := testCase.payment
			if payment == (model.AetherPayment{}) && testCase.name != "invalid payment" {
				payment = model.AetherPayment{Aes: 1, NonElemental: 1}
			}

			err := ValidateCastServant(
				&state,
				catalog,
				"player-one",
				"servant-one",
				payment,
				model.OrientationRecovered,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("ValidateCastServant() error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func TestValidateCastServantAcceptsRecoveredAndReversedEntry(t *testing.T) {
	for _, orientation := range []model.CardOrientation{
		model.OrientationRecovered,
		model.OrientationReversed,
	} {
		state, catalog := servantCastStateForTest()

		if err := ValidateCastServant(
			&state,
			catalog,
			"player-one",
			"servant-one",
			model.AetherPayment{Aes: 1, NonElemental: 1},
			orientation,
		); err != nil {
			t.Fatalf("ValidateCastServant() rejected orientation %q: %v", orientation, err)
		}
	}
}

func TestValidateCastServantRejectsInvalidEntryOrientation(t *testing.T) {
	for _, orientation := range []model.CardOrientation{
		"",
		model.OrientationRested,
		"sideways",
	} {
		state, catalog := servantCastStateForTest()

		err := ValidateCastServant(
			&state,
			catalog,
			"player-one",
			"servant-one",
			model.AetherPayment{Aes: 1, NonElemental: 1},
			orientation,
		)
		if err == nil || !strings.Contains(err.Error(), "recovered or reversed") {
			t.Fatalf("ValidateCastServant() orientation %q error = %v", orientation, err)
		}
	}
}

func TestValidateCastConjureAndBarrierAcceptEveryElement(t *testing.T) {
	type castValidator func(
		*model.MatchState,
		CardCatalog,
		model.PlayerID,
		model.MatchCardID,
		model.AetherPayment,
	) error

	casts := []struct {
		name     string
		cardType string
		validate castValidator
	}{
		{name: "Conjure", cardType: "Conjure", validate: ValidateCastConjure},
		{name: "Barrier", cardType: "Barrier", validate: ValidateCastBarrier},
	}
	elements := []struct {
		name    string
		value   string
		payment model.AetherPayment
	}{
		{name: "Aes", value: "Aes", payment: model.AetherPayment{Aes: 1, NonElemental: 1}},
		{name: "Aqua", value: " aqua ", payment: model.AetherPayment{Aqua: 1, NonElemental: 1}},
		{name: "Ignus", value: "IGNUS", payment: model.AetherPayment{Ignus: 1, NonElemental: 1}},
		{name: "Luna", value: "Luna", payment: model.AetherPayment{Luna: 1, NonElemental: 1}},
		{name: "Silva", value: "Silva", payment: model.AetherPayment{Silva: 1, NonElemental: 1}},
		{name: "Solis", value: "Solis", payment: model.AetherPayment{Solis: 1, NonElemental: 1}},
		{name: "Terra", value: "Terra", payment: model.AetherPayment{Terra: 1, NonElemental: 1}},
		{name: "Void", value: "Void", payment: model.AetherPayment{Void: 1, NonElemental: 1}},
	}

	for _, cast := range casts {
		for _, element := range elements {
			t.Run(cast.name+"/"+element.name, func(t *testing.T) {
				state, catalog := servantCastStateForTest()
				definition := catalog["printed-servant"]
				definition.Type = " " + strings.ToUpper(cast.cardType) + " "
				definition.Element = element.value
				catalog[definition.ID] = definition

				if err := cast.validate(
					&state,
					catalog,
					"player-one",
					"servant-one",
					element.payment,
				); err != nil {
					t.Fatalf("validator rejected valid %s: %v", cast.cardType, err)
				}
			})
		}
	}
}

func TestValidateCastConjureAndBarrierRejectCopiedEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cardType string
		validate func(*model.MatchState, CardCatalog, model.PlayerID, model.MatchCardID, model.AetherPayment) error
		mutate   func(*model.MatchState, definitionCatalogForTest)
		payment  model.AetherPayment
		wantErr  string
	}{
		{
			name:     "Conjure wrong type",
			cardType: "Conjure",
			validate: ValidateCastConjure,
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.Type = "Barrier"
				catalog[definition.ID] = definition
			},
			payment: model.AetherPayment{Aes: 1, NonElemental: 1},
			wantErr: "must be a conjure",
		},
		{
			name:     "Conjure wrong owner",
			cardType: "Conjure",
			validate: ValidateCastConjure,
			mutate: func(state *model.MatchState, _ definitionCatalogForTest) {
				instance := state.CardInstances["servant-one"]
				instance.Owner = "player-two"
				state.CardInstances["servant-one"] = instance
			},
			payment: model.AetherPayment{Aes: 1, NonElemental: 1},
			wantErr: "conjure being cast",
		},
		{
			name:     "Barrier wrong type",
			cardType: "Barrier",
			validate: ValidateCastBarrier,
			mutate: func(_ *model.MatchState, catalog definitionCatalogForTest) {
				definition := catalog["printed-servant"]
				definition.Type = "Conjure"
				catalog[definition.ID] = definition
			},
			payment: model.AetherPayment{Aes: 1, NonElemental: 1},
			wantErr: "must be a barrier",
		},
		{
			name:     "Barrier invalid payment",
			cardType: "Barrier",
			validate: ValidateCastBarrier,
			payment:  model.AetherPayment{Aes: 1},
			wantErr:  "error paying for barrier",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, catalog := servantCastStateForTest()
			definition := catalog["printed-servant"]
			definition.Type = testCase.cardType
			catalog[definition.ID] = definition
			if testCase.mutate != nil {
				testCase.mutate(&state, catalog)
			}

			err := testCase.validate(
				&state,
				catalog,
				"player-one",
				"servant-one",
				testCase.payment,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("validator error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func servantCastStateForTest() (model.MatchState, definitionCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant-one": {
				CardID:       "printed-servant",
				MatchID:      "servant-one",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: []model.MatchCardID{"servant-one"},
				Aether: model.AetherPool{
					Aes: 1, Aqua: 1, Ignus: 1, Luna: 1,
					Silva: 1, Solis: 1, Terra: 1, Void: 1,
					NonElemental: 1,
				},
			},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseMain,
		},
		PriorityHolder: "player-one",
		NextLinkID:     1,
	}
	catalog := definitionCatalogForTest{
		"printed-servant": gamecards.Card{
			ID:        "printed-servant",
			Type:      " Servant ",
			Element:   "Aes",
			CostLevel: "2",
		},
	}
	return state, catalog
}
