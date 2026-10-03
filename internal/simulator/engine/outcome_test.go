package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestFinishMatchRecordsDecisiveResultAndClosesPriority(t *testing.T) {
	state := finishMatchStateForTest()
	state.PrioritySequenceOpen = true
	state.PriorityHolder = "player-one"
	state.PassCount = 1
	result := model.MatchResult{
		Winner: "player-two",
		Loser:  "player-one",
		Reason: model.EndReasonDeckOut,
	}

	if err := finishMatch(&state, result); err != nil {
		t.Fatalf("finishMatch() error = %v", err)
	}
	if state.MatchStatus != model.StatusFinished {
		t.Fatalf("MatchStatus = %q; want %q", state.MatchStatus, model.StatusFinished)
	}
	if state.Result != result {
		t.Fatalf("Result = %#v; want %#v", state.Result, result)
	}
	if state.PrioritySequenceOpen || state.PriorityHolder != "" || state.PassCount != 0 {
		t.Fatalf(
			"priority after finish = open %t, holder %q, passes %d; want closed blank 0",
			state.PrioritySequenceOpen,
			state.PriorityHolder,
			state.PassCount,
		)
	}
	if state.Revision != 9 {
		t.Fatalf("Revision = %d; finishMatch must not increment it", state.Revision)
	}
}

func TestFinishMatchRecordsSimultaneousLossDraw(t *testing.T) {
	state := finishMatchStateForTest()
	result := model.MatchResult{
		IsDraw: true,
		Reason: model.EndReasonSimultaneousLoss,
	}

	if err := finishMatch(&state, result); err != nil {
		t.Fatalf("finishMatch() error = %v", err)
	}
	if state.MatchStatus != model.StatusFinished || !state.Result.IsDraw {
		t.Fatalf("finished draw = status %q, result %#v", state.MatchStatus, state.Result)
	}
	if state.Result.Winner != "" || state.Result.Loser != "" {
		t.Fatalf("draw named a winner or loser: %#v", state.Result)
	}
}

func TestFinishMatchRejectsInvalidResultWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*model.MatchState)
		result  model.MatchResult
		wantErr string
	}{
		{
			name: "match not in progress",
			mutate: func(state *model.MatchState) {
				state.MatchStatus = model.StatusFinished
			},
			result: model.MatchResult{
				Winner: "player-one",
				Loser:  "player-two",
				Reason: model.EndReasonZeroOrbs,
			},
			wantErr: "must be in",
		},
		{
			name: "unsupported reason",
			result: model.MatchResult{
				Winner: "player-one",
				Loser:  "player-two",
				Reason: "Concession",
			},
			wantErr: "unsupported match-end reason",
		},
		{
			name: "draw with wrong reason",
			result: model.MatchResult{
				IsDraw: true,
				Reason: model.EndReasonDeckOut,
			},
			wantErr: "draw result must use simultaneous-loss reason",
		},
		{
			name: "draw names a winner",
			result: model.MatchResult{
				Winner: "player-one",
				IsDraw: true,
				Reason: model.EndReasonSimultaneousLoss,
			},
			wantErr: "draw result cannot name a winner or loser",
		},
		{
			name: "simultaneous loss without draw",
			result: model.MatchResult{
				Winner: "player-one",
				Loser:  "player-two",
				Reason: model.EndReasonSimultaneousLoss,
			},
			wantErr: "simultaneous-loss reason must be a draw",
		},
		{
			name: "same winner and loser",
			result: model.MatchResult{
				Winner: "player-one",
				Loser:  "player-one",
				Reason: model.EndReasonDeckOut,
			},
			wantErr: "winner and loser must be distinct match players",
		},
		{
			name: "unknown winner",
			result: model.MatchResult{
				Winner: "spectator",
				Loser:  "player-two",
				Reason: model.EndReasonZeroOrbs,
			},
			wantErr: "winner and loser must be players in this match",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			state := finishMatchStateForTest()
			if testCase.mutate != nil {
				testCase.mutate(&state)
			}
			before := state

			err := finishMatch(&state, testCase.result)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("finishMatch() error = %v; want containing %q", err, testCase.wantErr)
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatalf("rejected finishMatch mutated state\n before: %#v\n  after: %#v", before, state)
			}
		})
	}
}

func TestFinishMatchRejectsNilState(t *testing.T) {
	err := finishMatch(nil, model.MatchResult{
		Winner: "player-one",
		Loser:  "player-two",
		Reason: model.EndReasonDeckOut,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("finishMatch(nil) error = %v; want nil-state error", err)
	}
}

func finishMatchStateForTest() model.MatchState {
	return model.MatchState{
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		MatchStatus:    model.StatusInProgress,
		Revision:       9,
		PriorityHolder: "player-one",
	}
}
