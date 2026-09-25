package engine

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestAddChaseLinkAppendsLinkAndTransfersPriority(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PassCount = 1

	err := AddChaseLink(
		&state,
		"player-one",
		"source-one",
		model.ChaseLinkCardPlay,
		state.Revision,
	)
	if err != nil {
		t.Fatalf("AddChaseLink() error = %v", err)
	}

	wantLink := model.ChaseLink{
		ID:           4,
		Controller:   "player-one",
		SourceCardID: "source-one",
		Kind:         model.ChaseLinkCardPlay,
	}
	if len(state.ChaseLinks) != 1 || state.ChaseLinks[0] != wantLink {
		t.Fatalf("ChaseLinks = %#v; want %#v", state.ChaseLinks, model.Chase{wantLink})
	}
	if state.NextLinkID != 5 {
		t.Fatalf("NextLinkID = %d; want 5", state.NextLinkID)
	}
	if state.PassCount != 0 {
		t.Fatalf("PassCount = %d; want 0", state.PassCount)
	}
	if state.PriorityHolder != "player-two" {
		t.Fatalf("PriorityHolder = %q; want %q", state.PriorityHolder, "player-two")
	}
	if state.Revision != 10 {
		t.Fatalf("Revision = %d; want 10", state.Revision)
	}
}

func TestAddChaseLinkRejectsStaleRevisionWithoutMutation(t *testing.T) {
	state := chaseEngineStateForTest()
	before := cloneChaseEngineState(state)

	err := AddChaseLink(
		&state,
		"player-one",
		"source-one",
		model.ChaseLinkCardPlay,
		state.Revision-1,
	)
	if err == nil || !strings.Contains(err.Error(), "state expected") {
		t.Fatalf("AddChaseLink() error = %v; want stale-revision error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated after stale revision:\n got: %#v\nwant: %#v", state, before)
	}
}

func TestAddChaseLinkRejectsRuleFailureWithoutMutation(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PriorityHolder = "player-two"
	before := cloneChaseEngineState(state)

	err := AddChaseLink(
		&state,
		"player-one",
		"source-one",
		model.ChaseLinkCardPlay,
		state.Revision,
	)
	if err == nil || !strings.Contains(err.Error(), "does not hold priority") {
		t.Fatalf("AddChaseLink() error = %v; want priority error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated after rule rejection:\n got: %#v\nwant: %#v", state, before)
	}
}

func TestAddChaseLinkRejectsMissingOpponentWithoutMutation(t *testing.T) {
	state := chaseEngineStateForTest()
	state.Players[1].ID = "player-one"
	before := cloneChaseEngineState(state)

	err := AddChaseLink(
		&state,
		"player-one",
		"source-one",
		model.ChaseLinkCardPlay,
		state.Revision,
	)
	if err == nil || !strings.Contains(err.Error(), "opposing player was not found") {
		t.Fatalf("AddChaseLink() error = %v; want missing-opponent error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated after missing opponent:\n got: %#v\nwant: %#v", state, before)
	}
}

func TestPassPriorityRecordsFirstPassAndTransfersPriority(t *testing.T) {
	state := chaseEngineStateForTest()

	err := PassPriority(&state, nil, "player-one", state.Revision)
	if err != nil {
		t.Fatalf("PassPriority() error = %v", err)
	}
	if state.PassCount != 1 {
		t.Fatalf("PassCount = %d; want 1", state.PassCount)
	}
	if state.PriorityHolder != "player-two" {
		t.Fatalf("PriorityHolder = %q; want %q", state.PriorityHolder, "player-two")
	}
	if state.Revision != 10 {
		t.Fatalf("Revision = %d; want 10", state.Revision)
	}
}

func TestPassPriorityResolvesTopLinkOnSecondPass(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PassCount = 1
	state.PriorityHolder = "player-two"
	state.ChaseLinks = model.Chase{{
		ID:               4,
		Controller:       "player-one",
		SourceCardID:     "source-one",
		Kind:             model.ChaseLinkCardPlay,
		EntryOrientation: model.OrientationRecovered,
	}}
	catalog := casterAetherCatalogForTest{
		"printed-one": gamecards.Card{ID: "printed-one", Type: "Servant"},
	}

	if err := PassPriority(&state, catalog, "player-two", state.Revision); err != nil {
		t.Fatalf("PassPriority() error = %v", err)
	}
	if len(state.ChaseLinks) != 0 {
		t.Fatalf("ChaseLinks = %#v; want empty Chase", state.ChaseLinks)
	}
	if state.PassCount != 0 || state.PriorityHolder != "player-one" {
		t.Fatalf("priority state = holder %q, passes %d; want player-one, 0", state.PriorityHolder, state.PassCount)
	}
	if state.Revision != 10 {
		t.Fatalf("Revision = %d; want 10", state.Revision)
	}
}

func TestPassPriorityRejectsUnimplementedEmptySecondPassWithoutMutation(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PassCount = 1
	state.PriorityHolder = "player-two"
	before := cloneChaseEngineState(state)

	err := PassPriority(&state, nil, "player-two", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "second pass not currently implemented") {
		t.Fatalf("PassPriority() error = %v; want unimplemented-second-pass error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated after rejected second pass:\n got: %#v\nwant: %#v", state, before)
	}
}

func TestPassPriorityPreservesStateWhenTopLinkResolutionFails(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PassCount = 1
	state.PriorityHolder = "player-two"
	state.ChaseLinks = model.Chase{{
		ID:           4,
		Controller:   "player-one",
		SourceCardID: "source-one",
		Kind:         model.ChaseLinkActivatedAbility,
	}}
	before := cloneChaseEngineState(state)

	err := PassPriority(&state, nil, "player-two", state.Revision)
	if err == nil || !strings.Contains(err.Error(), "activated ability resolution") {
		t.Fatalf("PassPriority() error = %v; want activated-ability resolution error", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated after failed resolution:\n got: %#v\nwant: %#v", state, before)
	}
}

func TestPassPriorityRejectsInvalidRequestWithoutMutation(t *testing.T) {
	tests := []struct {
		name             string
		mutate           func(*model.MatchState)
		expectedRevision func(model.MatchState) model.Revision
		wantErr          string
	}{
		{
			name: "stale revision",
			expectedRevision: func(state model.MatchState) model.Revision {
				return state.Revision - 1
			},
			wantErr: "state expected",
		},
		{
			name: "actor lacks priority",
			mutate: func(state *model.MatchState) {
				state.PriorityHolder = "player-two"
			},
			wantErr: "does not hold priority",
		},
		{
			name: "missing opponent",
			mutate: func(state *model.MatchState) {
				state.Players[1].ID = "player-one"
			},
			wantErr: "opposing player was not found",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseEngineStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			expectedRevision := state.Revision
			if testCase.expectedRevision != nil {
				expectedRevision = testCase.expectedRevision(state)
			}
			before := cloneChaseEngineState(state)

			err := PassPriority(&state, nil, "player-one", expectedRevision)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("PassPriority() error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("state mutated after rejected pass:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func TestResolveConjureCardPlayMovesCardToGraveyard(t *testing.T) {
	state := chaseEngineStateForTest()
	link := model.ChaseLink{
		ID:           4,
		Controller:   "player-one",
		SourceCardID: "source-one",
		Kind:         model.ChaseLinkCardPlay,
	}
	instance := state.CardInstances[link.SourceCardID]

	if err := resolveConjureCardPlay(&state, link, instance); err != nil {
		t.Fatalf("resolveConjureCardPlay() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[0].Graveyard, []model.MatchCardID{"source-one"}) {
		t.Fatalf("Graveyard = %#v; want source-one", state.Players[0].Graveyard)
	}
	if len(state.Players[0].ServantZone) != 0 {
		t.Fatalf("ServantZone = %#v; Conjure must not enter the field", state.Players[0].ServantZone)
	}
	resolved := state.CardInstances[link.SourceCardID]
	if resolved.Face != model.CardFaceUp || resolved.Controller != "player-one" {
		t.Fatalf("resolved Conjure = %#v; want face-up and controlled by player-one", resolved)
	}
}

func TestResolveServantCardPlayAppliesDeclaredOrientation(t *testing.T) {
	for _, orientation := range []model.CardOrientation{
		model.OrientationRecovered,
		model.OrientationReversed,
	} {
		t.Run(string(orientation), func(t *testing.T) {
			state := chaseEngineStateForTest()
			link := model.ChaseLink{
				ID:               4,
				Controller:       "player-one",
				SourceCardID:     "source-one",
				Kind:             model.ChaseLinkCardPlay,
				EntryOrientation: orientation,
			}
			instance := state.CardInstances[link.SourceCardID]

			if err := resolveServantCardPlay(&state, link, instance); err != nil {
				t.Fatalf("resolveServantCardPlay() error = %v", err)
			}
			if !reflect.DeepEqual(state.Players[0].ServantZone, []model.MatchCardID{"source-one"}) {
				t.Fatalf("ServantZone = %#v; want source-one", state.Players[0].ServantZone)
			}
			resolved := state.CardInstances[link.SourceCardID]
			if resolved.Face != model.CardFaceUp ||
				resolved.Orientation != orientation ||
				resolved.Controller != "player-one" {
				t.Fatalf("resolved Servant = %#v; want face-up, %q, and controlled by player-one", resolved, orientation)
			}
		})
	}
}

func TestResolveServantCardPlayRejectsInvalidOrientationWithoutMutation(t *testing.T) {
	for _, orientation := range []model.CardOrientation{
		"",
		model.OrientationRested,
		"sideways",
	} {
		t.Run(string(orientation), func(t *testing.T) {
			state := chaseEngineStateForTest()
			link := model.ChaseLink{
				ID:               4,
				Controller:       "player-one",
				SourceCardID:     "source-one",
				Kind:             model.ChaseLinkCardPlay,
				EntryOrientation: orientation,
			}
			instance := state.CardInstances[link.SourceCardID]
			before := cloneChaseEngineState(state)
			before.Players[0].ServantZone = append([]model.MatchCardID(nil), state.Players[0].ServantZone...)

			err := resolveServantCardPlay(&state, link, instance)
			if err == nil || !strings.Contains(err.Error(), "Recovered or Reversed") {
				t.Fatalf("resolveServantCardPlay() orientation %q error = %v", orientation, err)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("state mutated after invalid orientation:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func TestResolveBarrierCardPlayMovesCardToPersistentField(t *testing.T) {
	state := chaseEngineStateForTest()
	link := model.ChaseLink{
		ID:           4,
		Controller:   "player-one",
		SourceCardID: "source-one",
		Kind:         model.ChaseLinkCardPlay,
	}
	instance := state.CardInstances[link.SourceCardID]

	if err := resolveBarrierCardPlay(&state, link, instance); err != nil {
		t.Fatalf("resolveBarrierCardPlay() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[0].ServantZone, []model.MatchCardID{"source-one"}) {
		t.Fatalf("persistent field zone = %#v; want source-one", state.Players[0].ServantZone)
	}
	if len(state.Players[0].Graveyard) != 0 {
		t.Fatalf("Graveyard = %#v; resolved Barrier must remain in play", state.Players[0].Graveyard)
	}
	resolved := state.CardInstances[link.SourceCardID]
	if resolved.Face != model.CardFaceUp ||
		resolved.Orientation != model.OrientationRecovered ||
		resolved.Controller != "player-one" {
		t.Fatalf("resolved Barrier = %#v; want face-up, recovered, and controlled by player-one", resolved)
	}
}

func TestCardPlayDestinationHelpersRejectInvalidDestinationWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*model.MatchState)
		resolve func(*model.MatchState, model.ChaseLink, model.CardInstance) error
		wantErr string
	}{
		{
			name: "Conjure already in graveyard",
			prepare: func(state *model.MatchState) {
				state.Players[0].Graveyard = []model.MatchCardID{"source-one"}
			},
			resolve: resolveConjureCardPlay,
			wantErr: "already in the graveyard",
		},
		{
			name: "Barrier already in persistent field",
			prepare: func(state *model.MatchState) {
				state.Players[0].ServantZone = []model.MatchCardID{"source-one"}
			},
			resolve: resolveBarrierCardPlay,
			wantErr: "already in the persistent field zone",
		},
		{
			name:    "missing controller",
			resolve: resolveBarrierCardPlay,
			wantErr: "controlling player",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseEngineStateForTest()
			link := model.ChaseLink{
				ID:           4,
				Controller:   "player-one",
				SourceCardID: "source-one",
				Kind:         model.ChaseLinkCardPlay,
			}
			if testCase.name == "missing controller" {
				link.Controller = "missing-player"
			}
			if testCase.prepare != nil {
				testCase.prepare(&state)
			}
			instance := state.CardInstances[link.SourceCardID]
			before := cloneChaseEngineState(state)
			for index := range state.Players {
				before.Players[index].ServantZone = append([]model.MatchCardID(nil), state.Players[index].ServantZone...)
				before.Players[index].Graveyard = append([]model.MatchCardID(nil), state.Players[index].Graveyard...)
			}

			err := testCase.resolve(&state, link, instance)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("resolver error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("state mutated after rejected destination:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func TestResolveTopChaseLinkResolvesOnlyTopLinkAndResetsPriority(t *testing.T) {
	state := chaseEngineStateForTest()
	state.PassCount = 1
	state.PriorityHolder = "player-two"
	lowerLink := model.ChaseLink{
		ID:           3,
		Controller:   "player-two",
		SourceCardID: "lower-source",
		Kind:         model.ChaseLinkActivatedAbility,
	}
	state.ChaseLinks = model.Chase{
		lowerLink,
		{
			ID:               4,
			Controller:       "player-one",
			SourceCardID:     "source-one",
			Kind:             model.ChaseLinkCardPlay,
			EntryOrientation: model.OrientationReversed,
		},
	}
	catalog := casterAetherCatalogForTest{
		"printed-one": gamecards.Card{ID: "printed-one", Type: "Servant"},
	}

	if err := resolveTopChaseLink(&state, catalog, "player-two"); err != nil {
		t.Fatalf("resolveTopChaseLink() error = %v", err)
	}
	if !reflect.DeepEqual(state.ChaseLinks, model.Chase{lowerLink}) {
		t.Fatalf("ChaseLinks = %#v; want only lower link %#v", state.ChaseLinks, lowerLink)
	}
	if state.PassCount != 0 {
		t.Fatalf("PassCount = %d; want 0", state.PassCount)
	}
	if state.PriorityHolder != state.Turn.ActivePlayer {
		t.Fatalf("PriorityHolder = %q; want active player %q", state.PriorityHolder, state.Turn.ActivePlayer)
	}
	if state.Revision != 9 {
		t.Fatalf("Revision = %d; private resolver must not increment it", state.Revision)
	}
	if !reflect.DeepEqual(state.Players[0].ServantZone, []model.MatchCardID{"source-one"}) {
		t.Fatalf("ServantZone = %#v; want resolved top card", state.Players[0].ServantZone)
	}
	resolved := state.CardInstances["source-one"]
	if resolved.Face != model.CardFaceUp || resolved.Orientation != model.OrientationReversed {
		t.Fatalf("resolved Servant = %#v; want face-up and Reversed", resolved)
	}
}

func TestResolveTopChaseLinkResolvesConjureAndBarrierDestinations(t *testing.T) {
	tests := []struct {
		name     string
		cardType string
		assert   func(*testing.T, model.MatchState)
	}{
		{
			name:     "Conjure goes to Graveyard",
			cardType: "Conjure",
			assert: func(t *testing.T, state model.MatchState) {
				t.Helper()
				if !reflect.DeepEqual(state.Players[0].Graveyard, []model.MatchCardID{"source-one"}) {
					t.Fatalf("Graveyard = %#v; want resolved Conjure", state.Players[0].Graveyard)
				}
				if len(state.Players[0].ServantZone) != 0 {
					t.Fatalf("persistent field = %#v; Conjure must not remain in play", state.Players[0].ServantZone)
				}
			},
		},
		{
			name:     "Barrier remains in persistent field",
			cardType: "Barrier",
			assert: func(t *testing.T, state model.MatchState) {
				t.Helper()
				if !reflect.DeepEqual(state.Players[0].ServantZone, []model.MatchCardID{"source-one"}) {
					t.Fatalf("persistent field = %#v; want resolved Barrier", state.Players[0].ServantZone)
				}
				if len(state.Players[0].Graveyard) != 0 {
					t.Fatalf("Graveyard = %#v; Barrier must remain in play", state.Players[0].Graveyard)
				}
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseEngineStateForTest()
			state.PassCount = 1
			state.PriorityHolder = "player-two"
			state.ChaseLinks = model.Chase{{
				ID:           4,
				Controller:   "player-one",
				SourceCardID: "source-one",
				Kind:         model.ChaseLinkCardPlay,
			}}
			catalog := casterAetherCatalogForTest{
				"printed-one": gamecards.Card{ID: "printed-one", Type: testCase.cardType},
			}

			if err := resolveTopChaseLink(&state, catalog, "player-two"); err != nil {
				t.Fatalf("resolveTopChaseLink() error = %v", err)
			}
			if len(state.ChaseLinks) != 0 || state.PassCount != 0 || state.PriorityHolder != "player-one" {
				t.Fatalf("post-resolution Chase state = links %#v, passes %d, holder %q", state.ChaseLinks, state.PassCount, state.PriorityHolder)
			}
			if state.Revision != 9 {
				t.Fatalf("Revision = %d; private resolver must not increment it", state.Revision)
			}
			resolved := state.CardInstances["source-one"]
			if resolved.Face != model.CardFaceUp || resolved.Controller != "player-one" {
				t.Fatalf("resolved card = %#v; want face-up and controlled by player-one", resolved)
			}
			testCase.assert(t, state)
		})
	}
}

func TestResolveTopChaseLinkRejectsUnsupportedResolutionWithoutMutation(t *testing.T) {
	tests := []struct {
		name     string
		kind     model.ChaseLinkKind
		cardType string
		mutate   func(*model.MatchState)
		wantErr  string
	}{
		{
			name:     "missing active player",
			kind:     model.ChaseLinkCardPlay,
			cardType: "Servant",
			mutate: func(state *model.MatchState) {
				state.Turn.ActivePlayer = "missing-player"
			},
			wantErr: "active player",
		},
		{
			name:    "activated ability",
			kind:    model.ChaseLinkActivatedAbility,
			wantErr: "activated ability resolution",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseEngineStateForTest()
			state.PassCount = 1
			state.PriorityHolder = "player-two"
			state.ChaseLinks = model.Chase{{
				ID:               4,
				Controller:       "player-one",
				SourceCardID:     "source-one",
				Kind:             testCase.kind,
				EntryOrientation: model.OrientationRecovered,
			}}
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			catalog := casterAetherCatalogForTest{
				"printed-one": gamecards.Card{ID: "printed-one", Type: testCase.cardType},
			}
			before := cloneChaseEngineState(state)

			err := resolveTopChaseLink(&state, catalog, "player-two")
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("resolveTopChaseLink() error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("state mutated after rejected resolution:\n got: %#v\nwant: %#v", state, before)
			}
		})
	}
}

func chaseEngineStateForTest() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"source-one": {
				CardID:       "printed-one",
				MatchID:      "source-one",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		MatchStatus:    model.StatusInProgress,
		Revision:       9,
		Turn:           model.TurnState{Number: 1, ActivePlayer: "player-one", Phase: model.PhaseMain},
		PriorityHolder: "player-one",
		NextLinkID:     4,
	}
}

func cloneChaseEngineState(state model.MatchState) model.MatchState {
	clone := state
	clone.ChaseLinks = append(model.Chase(nil), state.ChaseLinks...)
	clone.CardInstances = make(map[model.MatchCardID]model.CardInstance, len(state.CardInstances))
	for id, instance := range state.CardInstances {
		clone.CardInstances[id] = instance
	}
	for index := range state.Players {
		clone.Players[index].ServantZone = append([]model.MatchCardID(nil), state.Players[index].ServantZone...)
		clone.Players[index].Graveyard = append([]model.MatchCardID(nil), state.Players[index].Graveyard...)
	}
	return clone
}
