package rules

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateUseCasterTokenAllowsEitherPlayerWithoutMutation(t *testing.T) {
	state := validCasterTokenRuleState()
	before := cloneCasterTokenRuleState(state)

	if err := ValidateUseCasterToken(&state, "player-two", "p2-token"); err != nil {
		t.Fatalf("ValidateUseCasterToken() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatalf("validation mutated state\n before: %#v\n  after: %#v", before, state)
	}
}

func TestValidateUseCasterTokenRejectsIllegalActionsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		playerID    model.PlayerID
		tokenID     model.MatchCardID
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{name: "match not in progress", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			state.MatchStatus = model.StatusSetup
		}, wantErrPart: "must be \"In Progress\""},
		{name: "unknown player", playerID: "missing-player", tokenID: "p2-token", wantErrPart: "not found in players"},
		{name: "token outside Caster Zone", playerID: "player-two", tokenID: "p1-token", wantErrPart: "not found in caster zone"},
		{name: "missing token instance", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			delete(state.CardInstances, "p2-token")
		}, wantErrPart: "not found"},
		{name: "wrong card definition", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.CardID = "printed-card"
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "not a caster token"},
		{name: "wrong card category", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.CardCategory = model.CategoryPrintedCard
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "not a token"},
		{name: "wrong owner", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.Owner = "player-one"
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "owner"},
		{name: "wrong controller", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.Controller = "player-one"
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "controller"},
		{name: "Rested token", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.Orientation = model.OrientationRested
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "must be recovered"},
		{name: "face-down token", playerID: "player-two", tokenID: "p2-token", mutate: func(state *model.MatchState) {
			instance := state.CardInstances["p2-token"]
			instance.Face = model.CardFaceDown
			state.CardInstances["p2-token"] = instance
		}, wantErrPart: "must be face-up"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := validCasterTokenRuleState()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := cloneCasterTokenRuleState(state)

			err := ValidateUseCasterToken(&state, testCase.playerID, testCase.tokenID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ValidateUseCasterToken() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected validation mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestValidateUseCasterTokenRejectsNilState(t *testing.T) {
	err := ValidateUseCasterToken(nil, "player-one", "p1-token")
	if err == nil || !strings.Contains(err.Error(), "state cannot be nil") {
		t.Fatalf("ValidateUseCasterToken(nil) error = %v; want nil-state error", err)
	}
}

func validCasterTokenRuleState() model.MatchState {
	return model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-token": casterTokenForTest("p1-token", "player-one"),
			"p2-token": casterTokenForTest("p2-token", "player-two"),
		},
		Players: [2]model.PlayerState{
			{ID: "player-one", CasterZone: []model.MatchCardID{"p1-token"}},
			{ID: "player-two", CasterZone: []model.MatchCardID{"p2-token"}},
		},
		MatchStatus: model.StatusInProgress,
		Turn:        model.TurnState{Number: 3, ActivePlayer: "player-one", Phase: model.PhaseBattle},
	}
}

func casterTokenForTest(matchID model.MatchCardID, playerID model.PlayerID) model.CardInstance {
	return model.CardInstance{
		CardID:       model.CasterTokenCardID,
		MatchID:      matchID,
		Owner:        playerID,
		Controller:   playerID,
		CardCategory: model.CategoryTokenCard,
		Face:         model.CardFaceUp,
		Orientation:  model.OrientationRecovered,
	}
}

func cloneCasterTokenRuleState(state model.MatchState) model.MatchState {
	clone := state
	clone.CardInstances = maps.Clone(state.CardInstances)
	for index := range state.Players {
		clone.Players[index].CasterZone = slices.Clone(state.Players[index].CasterZone)
		clone.Players[index].Exile = slices.Clone(state.Players[index].Exile)
	}
	return clone
}
