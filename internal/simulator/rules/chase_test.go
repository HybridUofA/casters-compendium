package rules

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateChasePriorityAcceptsEitherPlayerWhenTheyHoldPriority(t *testing.T) {
	state := chasePriorityStateForTest()
	state.Turn.ActivePlayer = state.Players[0].ID
	state.PriorityHolder = state.Players[1].ID

	if err := ValidateChasePriority(&state, state.Players[1].ID); err != nil {
		t.Fatalf("ValidateChasePriority() rejected non-active priority holder: %v", err)
	}
}

func TestValidateChasePriorityRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name    string
		state   func() *model.MatchState
		actor   model.PlayerID
		wantErr string
	}{
		{
			name:    "nil state",
			state:   func() *model.MatchState { return nil },
			actor:   "player-one",
			wantErr: "state cannot be nil",
		},
		{
			name: "match not in progress",
			state: func() *model.MatchState {
				state := chasePriorityStateForTest()
				state.MatchStatus = model.StatusSetup
				return &state
			},
			actor:   "player-one",
			wantErr: "expected",
		},
		{
			name:  "blank acting player",
			state: func() *model.MatchState { state := chasePriorityStateForTest(); return &state },
			actor: "  ", wantErr: "cannot be empty",
		},
		{
			name:  "unknown acting player",
			state: func() *model.MatchState { state := chasePriorityStateForTest(); return &state },
			actor: "spectator", wantErr: "acting player ID not found",
		},
		{
			name: "invalid priority holder",
			state: func() *model.MatchState {
				state := chasePriorityStateForTest()
				state.PriorityHolder = "spectator"
				return &state
			},
			actor: "player-one", wantErr: "neither player has priority",
		},
		{
			name:  "acting player lacks priority",
			state: func() *model.MatchState { state := chasePriorityStateForTest(); return &state },
			actor: "player-two", wantErr: "does not hold priority",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateChasePriority(testCase.state(), testCase.actor)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("ValidateChasePriority() error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func TestValidateAddChaseLinkAcceptsValidSource(t *testing.T) {
	state := chaseLinkStateForTest()

	err := ValidateAddChaseLink(
		&state,
		"player-one",
		"source-one",
		model.ChaseLinkCardPlay,
	)
	if err != nil {
		t.Fatalf("ValidateAddChaseLink() rejected valid link: %v", err)
	}
}

func TestValidateAddChaseLinkRejectsInvalidLink(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*model.MatchState)
		sourceID model.MatchCardID
		kind     model.ChaseLinkKind
		wantErr  string
	}{
		{
			name: "actor lacks priority",
			mutate: func(state *model.MatchState) {
				state.PriorityHolder = "player-two"
			},
			sourceID: "source-one",
			kind:     model.ChaseLinkCardPlay,
			wantErr:  "does not hold priority",
		},
		{
			name:     "invalid kind",
			sourceID: "source-one",
			kind:     "unknown",
			wantErr:  "invalid Chase link kind",
		},
		{
			name:     "blank source",
			sourceID: "  ",
			kind:     model.ChaseLinkCardPlay,
			wantErr:  "must not be empty",
		},
		{
			name:     "missing source",
			sourceID: "missing",
			kind:     model.ChaseLinkCardPlay,
			wantErr:  "was not found",
		},
		{
			name: "source controlled by other player",
			mutate: func(state *model.MatchState) {
				source := state.CardInstances["source-one"]
				source.Controller = "player-two"
				state.CardInstances["source-one"] = source
			},
			sourceID: "source-one",
			kind:     model.ChaseLinkActivatedAbility,
			wantErr:  "not the player activating",
		},
		{
			name: "zero next link ID",
			mutate: func(state *model.MatchState) {
				state.NextLinkID = 0
			},
			sourceID: "source-one",
			kind:     model.ChaseLinkTriggeredAbility,
			wantErr:  "must be positive",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseLinkStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}

			err := ValidateAddChaseLink(
				&state,
				"player-one",
				testCase.sourceID,
				testCase.kind,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("ValidateAddChaseLink() error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func TestValidatePassPriorityAcceptsValidPassCounts(t *testing.T) {
	for _, passCount := range []int{0, 1} {
		state := chaseLinkStateForTest()
		state.PassCount = passCount

		if err := ValidatePassPriority(&state, "player-one"); err != nil {
			t.Fatalf("ValidatePassPriority() rejected pass count %d: %v", passCount, err)
		}
	}
}

func TestValidatePassPriorityRejectsInvalidPassCounts(t *testing.T) {
	for _, passCount := range []int{-1, 2} {
		state := chaseLinkStateForTest()
		state.PassCount = passCount

		err := ValidatePassPriority(&state, "player-one")
		if err == nil || !strings.Contains(err.Error(), "must be between 0 and 1") {
			t.Fatalf("ValidatePassPriority() with pass count %d error = %v", passCount, err)
		}
	}
}

func TestValidateResolveTopChaseLinkAcceptsSecondPassWithLink(t *testing.T) {
	state := chaseLinkStateForTest()
	state.PassCount = 1
	state.ChaseLinks = model.Chase{
		{
			ID:           1,
			Controller:   "player-one",
			SourceCardID: "source-one",
			Kind:         model.ChaseLinkCardPlay,
		},
	}

	if err := ValidateResolveTopChaseLink(&state, "player-one"); err != nil {
		t.Fatalf("ValidateResolveTopChaseLink() rejected valid resolution: %v", err)
	}
}

func TestValidateResolveTopChaseLinkRejectsInvalidResolution(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*model.MatchState)
		wantErr string
	}{
		{
			name: "actor lacks priority",
			mutate: func(state *model.MatchState) {
				state.PriorityHolder = "player-two"
			},
			wantErr: "does not hold priority",
		},
		{
			name: "no prior pass",
			mutate: func(state *model.MatchState) {
				state.PassCount = 0
			},
			wantErr: "pass count must be 1",
		},
		{
			name: "empty chase",
			mutate: func(state *model.MatchState) {
				state.ChaseLinks = nil
			},
			wantErr: "empty chase",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := chaseLinkStateForTest()
			state.PassCount = 1
			state.ChaseLinks = model.Chase{
				{
					ID:           1,
					Controller:   "player-one",
					SourceCardID: "source-one",
					Kind:         model.ChaseLinkCardPlay,
				},
			}
			testCase.mutate(&state)

			err := ValidateResolveTopChaseLink(&state, "player-one")
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("ValidateResolveTopChaseLink() error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func chasePriorityStateForTest() model.MatchState {
	return model.MatchState{
		MatchStatus: model.StatusInProgress,
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		Turn:           model.TurnState{ActivePlayer: "player-one", Number: 1, Phase: model.PhaseMain},
		PriorityHolder: "player-one",
	}
}

func chaseLinkStateForTest() model.MatchState {
	state := chasePriorityStateForTest()
	state.NextLinkID = 1
	state.CardInstances = map[model.MatchCardID]model.CardInstance{
		"source-one": {
			CardID:       "printed-one",
			MatchID:      "source-one",
			Owner:        "player-one",
			Controller:   "player-one",
			CardCategory: model.CategoryPrintedCard,
		},
	}
	return state
}
