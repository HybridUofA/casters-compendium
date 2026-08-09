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

func TestValidateFaceDownLevelOneCallAcceptsLegalActionWithoutMutation(t *testing.T) {
	state := validFaceDownLevelOneCallState()
	before := cloneCallTestState(state)

	if err := ValidateFaceDownLevelOneCall(&state, "player-one", "p1-hand-card"); err != nil {
		t.Fatalf("ValidateFaceDownLevelOneCall() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateFaceDownLevelOneCallRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name         string
		actingPlayer model.PlayerID
		cardID       model.MatchCardID
		mutate       func(*model.MatchState)
		wantErrPart  string
	}{
		{
			name:         "match is not in progress",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusSetup
			},
			wantErrPart: "expected \"In Progress\"",
		},
		{
			name:         "not in Call phase",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.Phase = model.PhaseMain
			},
			wantErrPart: "expected \"Call\"",
		},
		{
			name:         "blank acting player",
			actingPlayer: "  ",
			cardID:       "p1-hand-card",
			wantErrPart:  "player ID cannot be empty",
		},
		{
			name:         "acting player is not active",
			actingPlayer: "player-two",
			cardID:       "p2-hand-card",
			wantErrPart:  "is not the active player",
		},
		{
			name:         "Call action was already taken",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.CallActionTaken = true
			},
			wantErrPart: "call action has already been taken",
		},
		{
			name:         "active player is absent from players",
			actingPlayer: "missing-player",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				state.Turn.ActivePlayer = "missing-player"
			},
			wantErrPart: "acting player ID not found",
		},
		{
			name:         "card is not in active player's hand",
			actingPlayer: "player-one",
			cardID:       "p2-hand-card",
			wantErrPart:  "card not found in hand",
		},
		{
			name:         "hand card is absent from card instances",
			actingPlayer: "player-one",
			cardID:       "p1-hand-card",
			mutate: func(state *model.MatchState) {
				delete(state.CardInstances, "p1-hand-card")
			},
			wantErrPart: "not found in card instances",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := validFaceDownLevelOneCallState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneCallTestState(state)

			err := ValidateFaceDownLevelOneCall(
				&state,
				testCase.actingPlayer,
				testCase.cardID,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateFaceDownLevelOneCall() error = %v; want error containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateFaceDownLevelOneCallRejectsNilState(t *testing.T) {
	err := ValidateFaceDownLevelOneCall(nil, "player-one", "p1-hand-card")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateFaceDownLevelOneCall(nil) error = %v; want nil-state error", err)
	}
}

func TestValidateFaceUpLevelOneCallAcceptsLegalActionWithoutMutation(t *testing.T) {
	state, catalog := validFaceUpLevelOneCallState()
	before := cloneCallTestState(state)

	if err := ValidateFaceUpLevelOneCall(&state, catalog, "player-one", "selected-caster"); err != nil {
		t.Fatalf("ValidateFaceUpLevelOneCall() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateFaceUpLevelOneCallRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		playerID    model.PlayerID
		cardID      model.MatchCardID
		catalog     func(definitionCatalogForTest) CardCatalog
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{name: "match not in progress", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			state.MatchStatus = model.StatusSetup
		}, wantErrPart: "expected \"In Progress\""},
		{name: "not Call phase", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			state.Turn.Phase = model.PhaseMain
		}, wantErrPart: "expected \"Call\""},
		{name: "blank player", playerID: " ", cardID: "selected-caster", wantErrPart: "player ID cannot be empty"},
		{name: "player is not active", playerID: "player-two", cardID: "p2-hand-card", wantErrPart: "is not the active player"},
		{name: "Call already taken", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			state.Turn.CallActionTaken = true
		}, wantErrPart: "already been taken"},
		{name: "active player missing", playerID: "missing", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			state.Turn.ActivePlayer = "missing"
		}, wantErrPart: "acting player ID not found"},
		{name: "card not in hand", playerID: "player-one", cardID: "distinct-caster", wantErrPart: "card not found in hand"},
		{name: "selected instance missing", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			delete(state.CardInstances, "selected-caster")
		}, wantErrPart: "not found in card instances"},
		{name: "selected card owned by opponent", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["selected-caster"]
			instance.Owner = "player-two"
			state.CardInstances["selected-caster"] = instance
		}, wantErrPart: "is not owned"},
		{name: "selected card is a token", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["selected-caster"]
			instance.CardCategory = model.CategoryTokenCard
			state.CardInstances["selected-caster"] = instance
		}, wantErrPart: "not a printed card"},
		{name: "nil catalog", playerID: "player-one", cardID: "selected-caster", catalog: func(definitionCatalogForTest) CardCatalog {
			return nil
		}, wantErrPart: "catalog cannot be nil"},
		{name: "selected definition missing", playerID: "player-one", cardID: "selected-caster", catalog: func(definitionCatalogForTest) CardCatalog {
			return definitionCatalogForTest{}
		}, wantErrPart: "error resolving card definition"},
		{name: "selected definition is not Caster", playerID: "player-one", cardID: "selected-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["selected-definition"]
			card.Type = "Servant"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "is not a caster"},
		{name: "selected level malformed", playerID: "player-one", cardID: "selected-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["selected-definition"]
			card.CostLevel = "one"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "level is malformed"},
		{name: "selected card is not Level 1", playerID: "player-one", cardID: "selected-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["selected-definition"]
			card.CostLevel = "2"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "level must be 1"},
		{name: "zone instance missing", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			delete(state.CardInstances, "distinct-caster")
		}, wantErrPart: "card cannot be resolved"},
		{name: "noncanonical zone token", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["distinct-caster"]
			instance.CardCategory = model.CategoryTokenCard
			state.CardInstances["distinct-caster"] = instance
		}, wantErrPart: "not a printed card"},
		{name: "zone card controlled by opponent", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["distinct-caster"]
			instance.Controller = "player-two"
			state.CardInstances["distinct-caster"] = instance
		}, wantErrPart: "is not acting player"},
		{name: "zone card has invalid face", playerID: "player-one", cardID: "selected-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["distinct-caster"]
			instance.Face = "sideways"
			state.CardInstances["distinct-caster"] = instance
		}, wantErrPart: "invalid face"},
		{name: "zone definition missing", playerID: "player-one", cardID: "selected-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			delete(catalog, "distinct-definition")
			return catalog
		}, wantErrPart: "error resolving definition"},
		{name: "face-up zone card is not Caster", playerID: "player-one", cardID: "selected-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["distinct-definition"]
			card.Type = "Servant"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "invalid zone for its type"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, defaultCatalog := validFaceUpLevelOneCallState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			catalog := CardCatalog(defaultCatalog)
			if testCase.catalog != nil {
				catalog = testCase.catalog(defaultCatalog)
			}
			before := cloneCallTestState(state)

			err := ValidateFaceUpLevelOneCall(&state, catalog, testCase.playerID, testCase.cardID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateFaceUpLevelOneCall() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateFaceUpLevelOneCallRejectsIdentityConflicts(t *testing.T) {
	state, catalog := validFaceUpLevelOneCallState()
	selected := catalog["selected-definition"]
	selected.Name = " Aria "
	selected.Subname = " DAWN "
	selected.Traits = "[Mage / Aqua]"
	existing := catalog["distinct-definition"]
	existing.Name = "aria"
	existing.Subname = "dawn"
	existing.Traits = "[aqua/mage]"
	catalog[selected.ID] = selected
	catalog[existing.ID] = existing

	err := ValidateFaceUpLevelOneCall(&state, catalog, "player-one", "selected-caster")
	if err == nil || !strings.Contains(err.Error(), "already in zone") {
		t.Fatalf("ValidateFaceUpLevelOneCall() error = %v; want complete identity-conflict error", err)
	}
}

func TestCasterDefinitionsConflictRequiresCompleteIdentityMatch(t *testing.T) {
	tests := []struct {
		name     string
		selected gamecards.Card
		existing gamecards.Card
		want     bool
	}{
		{
			name:     "normalized complete identity",
			selected: gamecards.Card{Name: " Aria ", Subname: "Dawn", Traits: "[Mage / Aqua]"},
			existing: gamecards.Card{Name: "aria", Subname: " dawn ", Traits: "[aqua/mage]"},
			want:     true,
		},
		{name: "different name", selected: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage]"}, existing: gamecards.Card{Name: "Bryn", Subname: "Dawn", Traits: "[Mage]"}},
		{name: "different subname", selected: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage]"}, existing: gamecards.Card{Name: "Aria", Subname: "Dusk", Traits: "[Mage]"}},
		{name: "different complete traits", selected: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage/Aqua]"}, existing: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage/Fire]"}},
		{name: "only one overlapping trait", selected: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage/Aqua]"}, existing: gamecards.Card{Name: "Aria", Subname: "Dawn", Traits: "[Mage/Knight]"}},
		{name: "blank identities", selected: gamecards.Card{}, existing: gamecards.Card{}},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := casterDefinitionsConflict(testCase.selected, testCase.existing); got != testCase.want {
				t.Fatalf("casterDefinitionsConflict() = %t; want %t", got, testCase.want)
			}
		})
	}
}

func TestValidateLevelUpCasterAcceptsLegalActionWithoutMutation(t *testing.T) {
	state, catalog := validLevelUpCasterState()
	before := cloneCallTestState(state)

	if err := ValidateLevelUpCaster(&state, catalog, "player-one", "upper-caster", "target-caster"); err != nil {
		t.Fatalf("ValidateLevelUpCaster() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateLevelUpCasterRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		playerID    model.PlayerID
		upperID     model.MatchCardID
		targetID    model.MatchCardID
		catalog     func(definitionCatalogForTest) CardCatalog
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{name: "match not in progress", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { state.MatchStatus = model.StatusSetup }, wantErrPart: "expected \"In Progress\""},
		{name: "not Call phase", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { state.Turn.Phase = model.PhaseMain }, wantErrPart: "expected \"Call\""},
		{name: "blank player", playerID: " ", upperID: "upper-caster", targetID: "target-caster", wantErrPart: "player ID cannot be empty"},
		{name: "player not active", playerID: "player-two", upperID: "p2-hand-card", targetID: "target-caster", wantErrPart: "is not the active player"},
		{name: "Call already taken", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { state.Turn.CallActionTaken = true }, wantErrPart: "already been taken"},
		{name: "active player missing", playerID: "missing", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { state.Turn.ActivePlayer = "missing" }, wantErrPart: "acting player ID not found"},
		{name: "upper card not in hand", playerID: "player-one", upperID: "target-caster", targetID: "target-caster", wantErrPart: "card not found in hand"},
		{name: "upper instance missing", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { delete(state.CardInstances, "upper-caster") }, wantErrPart: "not found in card instances"},
		{name: "upper owned by opponent", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["upper-caster"]
			instance.Owner = "player-two"
			state.CardInstances["upper-caster"] = instance
		}, wantErrPart: "is not owned"},
		{name: "upper is token", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["upper-caster"]
			instance.CardCategory = model.CategoryTokenCard
			state.CardInstances["upper-caster"] = instance
		}, wantErrPart: "not a printed card"},
		{name: "upper hand card already has Stock", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["upper-caster"]
			instance.Stock = []model.MatchCardID{"invalid-stock"}
			state.CardInstances["upper-caster"] = instance
		}, wantErrPart: "cannot have Stock"},
		{name: "nil catalog", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(definitionCatalogForTest) CardCatalog { return nil }, wantErrPart: "catalog cannot be nil"},
		{name: "upper definition missing", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(definitionCatalogForTest) CardCatalog { return definitionCatalogForTest{} }, wantErrPart: "error resolving card definition"},
		{name: "upper is not Caster", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["upper-definition"]
			card.Type = "Servant"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "is not a caster"},
		{name: "upper level malformed", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["upper-definition"]
			card.CostLevel = "three"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "level is malformed"},
		{name: "target outside Caster Zone", playerID: "player-one", upperID: "upper-caster", targetID: "outside-target", mutate: func(state *model.MatchState) {
			state.CardInstances["outside-target"] = state.CardInstances["target-caster"]
		}, wantErrPart: "not in the acting player's Caster Zone"},
		{name: "target instance missing", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) { delete(state.CardInstances, "target-caster") }, wantErrPart: "card cannot be resolved"},
		{name: "target is token", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["target-caster"]
			instance.CardCategory = model.CategoryTokenCard
			state.CardInstances["target-caster"] = instance
		}, wantErrPart: "cannot be a token"},
		{name: "target controlled by opponent", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["target-caster"]
			instance.Controller = "player-two"
			state.CardInstances["target-caster"] = instance
		}, wantErrPart: "not controlled"},
		{name: "target face down", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["target-caster"]
			instance.Face = model.CardFaceDown
			state.CardInstances["target-caster"] = instance
		}, wantErrPart: "must be face-up"},
		{name: "target definition missing", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			delete(catalog, "target-definition")
			return catalog
		}, wantErrPart: "error resolving card definition"},
		{name: "target is not Caster", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["target-definition"]
			card.Type = "Servant"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "must be a caster"},
		{name: "different names", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["target-definition"]
			card.Name = "Bryn"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "must share a name"},
		{name: "blank names", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			upper := catalog["upper-definition"]
			upper.Name = " "
			catalog[upper.ID] = upper
			target := catalog["target-definition"]
			target.Name = ""
			catalog[target.ID] = target
			return catalog
		}, wantErrPart: "must share a name"},
		{name: "target level malformed", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["target-definition"]
			card.CostLevel = "two"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "parsing target caster level"},
		{name: "levels not consecutive", playerID: "player-one", upperID: "upper-caster", targetID: "target-caster", catalog: func(catalog definitionCatalogForTest) CardCatalog {
			card := catalog["upper-definition"]
			card.CostLevel = "4"
			catalog[card.ID] = card
			return catalog
		}, wantErrPart: "must be one level higher"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state, defaultCatalog := validLevelUpCasterState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			catalog := CardCatalog(defaultCatalog)
			if testCase.catalog != nil {
				catalog = testCase.catalog(defaultCatalog)
			}
			before := cloneCallTestState(state)

			err := ValidateLevelUpCaster(&state, catalog, testCase.playerID, testCase.upperID, testCase.targetID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateLevelUpCaster() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateLevelUpCasterRejectsNilState(t *testing.T) {
	err := ValidateLevelUpCaster(nil, definitionCatalogForTest{}, "player-one", "upper-caster", "target-caster")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateLevelUpCaster(nil) error = %v; want nil-state error", err)
	}
}

func TestValidateFaceUpLevelOneCallRejectsNilState(t *testing.T) {
	err := ValidateFaceUpLevelOneCall(nil, definitionCatalogForTest{}, "player-one", "selected-caster")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateFaceUpLevelOneCall(nil) error = %v; want nil-state error", err)
	}
}

func validFaceUpLevelOneCallState() (model.MatchState, definitionCatalogForTest) {
	state := validFaceDownLevelOneCallState()
	state.Players[0].Hand = []model.MatchCardID{"selected-caster"}
	state.Players[0].CasterZone = []model.MatchCardID{"caster-token", "face-down-caster", "distinct-caster"}
	state.CardInstances["selected-caster"] = model.CardInstance{
		CardID: "selected-definition", MatchID: "selected-caster", Owner: "player-one",
		Controller: "player-one", CardCategory: model.CategoryPrintedCard,
	}
	state.CardInstances["caster-token"] = model.CardInstance{
		CardID: model.CasterTokenCardID, MatchID: "caster-token", Owner: "player-one",
		Controller: "player-one", CardCategory: model.CategoryTokenCard,
		Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
	}
	state.CardInstances["face-down-caster"] = model.CardInstance{
		CardID: "conflicting-definition", MatchID: "face-down-caster", Owner: "player-one",
		Controller: "player-one", CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	state.CardInstances["distinct-caster"] = model.CardInstance{
		CardID: "distinct-definition", MatchID: "distinct-caster", Owner: "player-one",
		Controller: "player-one", CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp,
	}
	catalog := definitionCatalogForTest{
		"selected-definition":    {ID: "selected-definition", Name: "Aria", Subname: "Dawn", Type: "Caster", Traits: "[Mage/Aqua]", CostLevel: " 1 "},
		"conflicting-definition": {ID: "conflicting-definition", Name: "Aria", Subname: "Dawn", Type: "Caster", Traits: "[Mage/Aqua]", CostLevel: "1"},
		"distinct-definition":    {ID: "distinct-definition", Name: "Bryn", Subname: "Dusk", Type: "Caster", Traits: "[Knight/Fire]", CostLevel: "2"},
	}
	return state, catalog
}

func validLevelUpCasterState() (model.MatchState, definitionCatalogForTest) {
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"upper-caster": {
				CardID: "upper-definition", MatchID: "upper-caster", Owner: "player-one",
				Controller: "player-one", CardCategory: model.CategoryPrintedCard,
			},
			"target-caster": {
				CardID: "target-definition", MatchID: "target-caster", Owner: "player-one",
				Controller: "player-one", CardCategory: model.CategoryPrintedCard,
				Face: model.CardFaceUp, Orientation: model.OrientationRested,
				Stock: []model.MatchCardID{"level-one-stock"},
			},
			"level-one-stock": {
				CardID: "stock-definition", MatchID: "level-one-stock", Owner: "player-one",
				Controller: "player-one", CardCategory: model.CategoryPrintedCard,
				Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
			},
			"p2-hand-card": {
				CardID: "p2-definition", MatchID: "p2-hand-card", Owner: "player-two",
				Controller: "player-two", CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Hand: []model.MatchCardID{"upper-caster"}, CasterZone: []model.MatchCardID{"target-caster"}},
			{ID: "player-two", Hand: []model.MatchCardID{"p2-hand-card"}},
		},
		MatchStatus: model.StatusInProgress,
		Revision:    9,
		Turn: model.TurnState{
			Number: 3, ActivePlayer: "player-one", Phase: model.PhaseCall,
		},
	}
	catalog := definitionCatalogForTest{
		"upper-definition":  {ID: "upper-definition", Name: " Aria ", Subname: "Ascendant", Type: " CASTER ", Traits: "[Mage/Aqua]", CostLevel: " 3 "},
		"target-definition": {ID: "target-definition", Name: "aria", Subname: "Dawn", Type: "Caster", Traits: "[Mage]", CostLevel: "2"},
		"stock-definition":  {ID: "stock-definition", Name: "Aria", Type: "Caster", CostLevel: "1"},
		"p2-definition":     {ID: "p2-definition", Name: "Opponent", Type: "Caster", CostLevel: "3"},
	}
	return state, catalog
}

func validFaceDownLevelOneCallState() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-hand-card": {
				CardID:       "definition-one",
				MatchID:      "p1-hand-card",
				Owner:        "player-one",
				Controller:   "player-one",
				CardCategory: model.CategoryPrintedCard,
			},
			"p2-hand-card": {
				CardID:       "definition-two",
				MatchID:      "p2-hand-card",
				Owner:        "player-two",
				Controller:   "player-two",
				CardCategory: model.CategoryPrintedCard,
			},
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", Hand: []model.MatchCardID{"p1-hand-card"}},
			{ID: "player-two", Hand: []model.MatchCardID{"p2-hand-card"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn: model.TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        model.PhaseCall,
		},
	}
}

func cloneCallTestState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for matchID, instance := range clone.CardInstances {
		instance.Stock = slices.Clone(instance.Stock)
		clone.CardInstances[matchID] = instance
	}
	for index := range state.Players {
		clone.Players[index].Deck = slices.Clone(state.Players[index].Deck)
		clone.Players[index].Hand = slices.Clone(state.Players[index].Hand)
		clone.Players[index].Orbs = slices.Clone(state.Players[index].Orbs)
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
	}
	return clone
}
