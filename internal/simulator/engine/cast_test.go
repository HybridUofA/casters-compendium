package engine

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestCastServantPaysAndAddsCardToChase(t *testing.T) {
	state, catalog := castServantEngineStateForTest()
	payment := model.AetherPayment{Aes: 1, NonElemental: 2}

	err := CastServant(
		&state,
		catalog,
		"player-one",
		"servant-one",
		payment,
		model.OrientationReversed,
		state.Revision,
	)
	if err != nil {
		t.Fatalf("CastServant() error = %v", err)
	}

	wantPool := model.AetherPool{Aes: 1, NonElemental: 1}
	if state.Players[0].Aether != wantPool {
		t.Fatalf("Aether = %#v; want %#v", state.Players[0].Aether, wantPool)
	}
	if !reflect.DeepEqual(state.Players[0].Hand, []model.MatchCardID{"other-card"}) {
		t.Fatalf("Hand = %#v; want only other-card", state.Players[0].Hand)
	}
	if len(state.Players[0].ServantZone) != 0 {
		t.Fatalf("ServantZone = %#v; unresolved Servant must remain on Chase", state.Players[0].ServantZone)
	}

	instance := state.CardInstances["servant-one"]
	if instance.Face != model.CardFaceUp || instance.Controller != "player-one" {
		t.Fatalf("Servant instance = %#v; want face-up and controlled by player-one", instance)
	}
	wantLink := model.ChaseLink{
		ID:               4,
		Controller:       "player-one",
		SourceCardID:     "servant-one",
		Kind:             model.ChaseLinkCardPlay,
		EntryOrientation: model.OrientationReversed,
	}
	if len(state.ChaseLinks) != 1 || state.ChaseLinks[0] != wantLink {
		t.Fatalf("ChaseLinks = %#v; want %#v", state.ChaseLinks, model.Chase{wantLink})
	}
	if state.NextLinkID != 5 || state.PassCount != 0 {
		t.Fatalf("NextLinkID/PassCount = %d/%d; want 5/0", state.NextLinkID, state.PassCount)
	}
	if state.PriorityHolder != "player-two" {
		t.Fatalf("PriorityHolder = %q; want player-two", state.PriorityHolder)
	}
	if state.Revision != 10 {
		t.Fatalf("Revision = %d; want 10", state.Revision)
	}
}

func TestCastConjureAndBarrierUseSharedCastMutation(t *testing.T) {
	tests := []struct {
		name     string
		cardType string
		cast     func(*model.MatchState, casterAetherCatalogForTest, model.PlayerID, model.MatchCardID, model.AetherPayment, model.Revision) error
	}{
		{
			name:     "Conjure",
			cardType: "Conjure",
			cast: func(state *model.MatchState, catalog casterAetherCatalogForTest, playerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, revision model.Revision) error {
				return CastConjure(state, catalog, playerID, cardID, payment, revision)
			},
		},
		{
			name:     "Barrier",
			cardType: "Barrier",
			cast: func(state *model.MatchState, catalog casterAetherCatalogForTest, playerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, revision model.Revision) error {
				return CastBarrier(state, catalog, playerID, cardID, payment, revision)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, catalog := castServantEngineStateForTest()
			definition := catalog["printed-servant"]
			definition.Type = testCase.cardType
			catalog[definition.ID] = definition

			err := testCase.cast(
				&state,
				catalog,
				"player-one",
				"servant-one",
				model.AetherPayment{Aes: 1, NonElemental: 2},
				state.Revision,
			)
			if err != nil {
				t.Fatalf("cast %s error = %v", testCase.cardType, err)
			}
			if !reflect.DeepEqual(state.Players[0].Hand, []model.MatchCardID{"other-card"}) {
				t.Fatalf("Hand = %#v; want only other-card", state.Players[0].Hand)
			}
			if state.Players[0].Aether != (model.AetherPool{Aes: 1, NonElemental: 1}) {
				t.Fatalf("Aether = %#v; payment was not applied", state.Players[0].Aether)
			}
			if len(state.ChaseLinks) != 1 || state.ChaseLinks[0].Kind != model.ChaseLinkCardPlay {
				t.Fatalf("ChaseLinks = %#v; want one card-play link", state.ChaseLinks)
			}
			if state.PriorityHolder != "player-two" || state.Revision != 10 {
				t.Fatalf("priority/revision = %q/%d; want player-two/10", state.PriorityHolder, state.Revision)
			}
		})
	}
}

func TestCastServantRejectsInvalidRequestWithoutMutation(t *testing.T) {
	tests := []struct {
		name             string
		mutate           func(*model.MatchState)
		payment          model.AetherPayment
		expectedRevision func(model.MatchState) model.Revision
		wantErr          string
	}{
		{
			name:    "invalid payment",
			payment: model.AetherPayment{Aes: 1},
			wantErr: "error paying for servant",
		},
		{
			name:    "stale revision",
			payment: model.AetherPayment{Aes: 1, NonElemental: 2},
			expectedRevision: func(state model.MatchState) model.Revision {
				return state.Revision - 1
			},
			wantErr: "expected revision",
		},
		{
			name: "missing opponent",
			mutate: func(state *model.MatchState) {
				state.Players[1].ID = "player-one"
			},
			payment: model.AetherPayment{Aes: 1, NonElemental: 2},
			wantErr: "opposing player was not found",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, catalog := castServantEngineStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			expectedRevision := state.Revision
			if testCase.expectedRevision != nil {
				expectedRevision = testCase.expectedRevision(state)
			}
			before := cloneCastEngineState(state)

			err := CastServant(
				&state,
				catalog,
				"player-one",
				"servant-one",
				testCase.payment,
				model.OrientationRecovered,
				expectedRevision,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("CastServant() error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("state mutated after rejected cast:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func castServantEngineStateForTest() (model.MatchState, casterAetherCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant-one": {
				CardID:       "printed-servant",
				MatchID:      "servant-one",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
			"other-card": {
				CardID:       "printed-other",
				MatchID:      "other-card",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{
				ID:     "player-one",
				Hand:   []model.MatchCardID{"servant-one", "other-card"},
				Aether: model.AetherPool{Aes: 2, NonElemental: 3},
			},
			{ID: "player-two"},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    9,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseMain,
		},
		PriorityHolder: "player-one",
		PassCount:      1,
		NextLinkID:     4,
	}
	catalog := casterAetherCatalogForTest{
		"printed-servant": gamecards.Card{
			ID:        "printed-servant",
			Type:      "Servant",
			Element:   "Aes",
			CostLevel: "3",
		},
	}
	return state, catalog
}

func cloneCastEngineState(state model.MatchState) model.MatchState {
	clone := state
	clone.ChaseLinks = append(model.Chase(nil), state.ChaseLinks...)
	clone.CardInstances = make(map[model.MatchCardID]model.CardInstance, len(state.CardInstances))
	for id, instance := range state.CardInstances {
		clone.CardInstances[id] = instance
	}
	for index := range state.Players {
		clone.Players[index].Hand = append([]model.MatchCardID(nil), state.Players[index].Hand...)
	}
	return clone
}
