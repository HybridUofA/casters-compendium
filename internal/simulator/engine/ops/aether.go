package ops

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// AddAether adds amount of element to playerID's pool.
// element is an Aether key such as "Aes" or "NonElemental".
func AddAether(state *model.MatchState, playerID model.PlayerID, element string, amount int) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative: %d", amount)
	}
	return adjustAether(state, playerID, element, amount)
}

// SpendAether removes amount of element from playerID's pool.
func SpendAether(state *model.MatchState, playerID model.PlayerID, element string, amount int) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative: %d", amount)
	}
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	key, err := rules.ParseAetherElementKey(element)
	if err != nil {
		return err
	}
	if rules.AetherPoolAmount(state.Players[index].Aether, key) < amount {
		return fmt.Errorf("insufficient %s aether", element)
	}
	rules.ApplyAetherPoolDelta(&state.Players[index].Aether, key, -amount)
	return nil
}

func adjustAether(state *model.MatchState, playerID model.PlayerID, element string, delta int) error {
	index, err := playerIndex(state, playerID)
	if err != nil {
		return err
	}
	key, err := rules.ParseAetherElementKey(element)
	if err != nil {
		return err
	}
	rules.ApplyAetherPoolDelta(&state.Players[index].Aether, key, delta)
	return nil
}
