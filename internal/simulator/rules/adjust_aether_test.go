package rules

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateAdjustAetherAllowsPlusAndMinusWithoutMutation(t *testing.T) {
	state := model.MatchState{
		MatchStatus: model.StatusInProgress,
		Players: [2]model.PlayerState{
			{ID: "player-one", Aether: model.AetherPool{Aes: 1}},
			{ID: "player-two"},
		},
	}
	before := state.Players[0].Aether

	if err := ValidateAdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "Aes", Delta: 1}); err != nil {
		t.Fatalf("ValidateAdjustAether(+1) error = %v", err)
	}
	if err := ValidateAdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "Aes", Delta: -1}); err != nil {
		t.Fatalf("ValidateAdjustAether(-1) error = %v", err)
	}
	if state.Players[0].Aether != before {
		t.Fatal("ValidateAdjustAether mutated the pool")
	}
}

func TestValidateAdjustAetherRejectsIllegalCommands(t *testing.T) {
	state := model.MatchState{
		MatchStatus: model.StatusInProgress,
		Players: [2]model.PlayerState{
			{ID: "player-one", Aether: model.AetherPool{Void: 0}},
			{ID: "player-two"},
		},
	}
	tests := []struct {
		name    string
		command model.AdjustAetherCommand
		wantErr string
	}{
		{name: "bad delta", command: model.AdjustAetherCommand{Element: "Aes", Delta: 2}, wantErr: "delta must be +1 or -1"},
		{name: "unknown element", command: model.AdjustAetherCommand{Element: "Fire", Delta: 1}, wantErr: "unknown Aether element"},
		{name: "below zero", command: model.AdjustAetherCommand{Element: "Void", Delta: -1}, wantErr: "cannot reduce Void Aether below 0"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateAdjustAether(&state, "player-one", testCase.command)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v; want containing %q", err, testCase.wantErr)
			}
		})
	}
}
