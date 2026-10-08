package engine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestAdjustAetherAddsAndRemovesOwnPool(t *testing.T) {
	state := model.MatchState{
		MatchStatus: model.StatusInProgress,
		Revision:    3,
		Players: [2]model.PlayerState{
			{ID: "player-one", Aether: model.AetherPool{Aes: 1, NonElemental: 2}},
			{ID: "player-two", Aether: model.AetherPool{Void: 4}},
		},
	}

	if err := AdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "Aes", Delta: 1}, 3); err != nil {
		t.Fatalf("AdjustAether(+Aes) error = %v", err)
	}
	if state.Players[0].Aether.Aes != 2 || state.Revision != 4 {
		t.Fatalf("after +Aes: pool=%#v revision=%d", state.Players[0].Aether, state.Revision)
	}
	if err := AdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "NonElemental", Delta: -1}, 4); err != nil {
		t.Fatalf("AdjustAether(-NonElemental) error = %v", err)
	}
	if state.Players[0].Aether.NonElemental != 1 || state.Revision != 5 {
		t.Fatalf("after -NonElemental: pool=%#v revision=%d", state.Players[0].Aether, state.Revision)
	}
	if state.Players[1].Aether.Void != 4 {
		t.Fatal("AdjustAether mutated the opponent pool")
	}
}

func TestAdjustAetherRejectsWithoutMutation(t *testing.T) {
	state := model.MatchState{
		MatchStatus: model.StatusInProgress,
		Revision:    1,
		Players: [2]model.PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
	}
	before := state

	err := AdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "Aqua", Delta: -1}, 1)
	if err == nil || !strings.Contains(err.Error(), "cannot reduce Aqua Aether below 0") {
		t.Fatalf("error = %v; want below-zero rejection", err)
	}
	if !reflect.DeepEqual(state, before) {
		t.Fatal("rejected AdjustAether mutated state")
	}

	err = AdjustAether(&state, "player-one", model.AdjustAetherCommand{Element: "Aqua", Delta: 1}, 0)
	if err == nil || !strings.Contains(err.Error(), "expected revision") {
		t.Fatalf("error = %v; want stale revision", err)
	}
}
