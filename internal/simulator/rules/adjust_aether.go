package rules

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// AetherElementKey identifies one counter in a player's Aether pool.
type AetherElementKey string

const (
	AetherElementAes          AetherElementKey = "Aes"
	AetherElementAqua         AetherElementKey = "Aqua"
	AetherElementIgnus        AetherElementKey = "Ignus"
	AetherElementLuna         AetherElementKey = "Luna"
	AetherElementSilva        AetherElementKey = "Silva"
	AetherElementSolis        AetherElementKey = "Solis"
	AetherElementTerra        AetherElementKey = "Terra"
	AetherElementVoid         AetherElementKey = "Void"
	AetherElementNonElemental AetherElementKey = "NonElemental"
)

// ValidateAdjustAether checks a manual ±1 Aether pool adjustment for the acting player.
func ValidateAdjustAether(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	command model.AdjustAetherCommand,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if command.Delta != 1 && command.Delta != -1 {
		return fmt.Errorf("delta must be +1 or -1, got %d", command.Delta)
	}
	key, err := ParseAetherElementKey(command.Element)
	if err != nil {
		return err
	}
	playerIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("acting player ID not found")
	}
	amount := aetherPoolAmount(state.Players[playerIndex].Aether, key)
	if command.Delta < 0 && amount+command.Delta < 0 {
		return fmt.Errorf("cannot reduce %s Aether below 0", key)
	}
	return nil
}

// ParseAetherElementKey normalizes a command element string to a pool key.
func ParseAetherElementKey(raw string) (AetherElementKey, error) {
	switch strings.TrimSpace(raw) {
	case "Aes":
		return AetherElementAes, nil
	case "Aqua":
		return AetherElementAqua, nil
	case "Ignus":
		return AetherElementIgnus, nil
	case "Luna":
		return AetherElementLuna, nil
	case "Silva":
		return AetherElementSilva, nil
	case "Solis":
		return AetherElementSolis, nil
	case "Terra":
		return AetherElementTerra, nil
	case "Void":
		return AetherElementVoid, nil
	case "NonElemental", "Non-elemental", "non-elemental":
		return AetherElementNonElemental, nil
	default:
		return "", fmt.Errorf("unknown Aether element %q", raw)
	}
}

func aetherPoolAmount(pool model.AetherPool, key AetherElementKey) int {
	switch key {
	case AetherElementAes:
		return pool.Aes
	case AetherElementAqua:
		return pool.Aqua
	case AetherElementIgnus:
		return pool.Ignus
	case AetherElementLuna:
		return pool.Luna
	case AetherElementSilva:
		return pool.Silva
	case AetherElementSolis:
		return pool.Solis
	case AetherElementTerra:
		return pool.Terra
	case AetherElementVoid:
		return pool.Void
	case AetherElementNonElemental:
		return pool.NonElemental
	default:
		return 0
	}
}

// ApplyAetherPoolDelta mutates one pool counter by delta. Callers must validate first.
func ApplyAetherPoolDelta(pool *model.AetherPool, key AetherElementKey, delta int) {
	if pool == nil {
		return
	}
	switch key {
	case AetherElementAes:
		pool.Aes += delta
	case AetherElementAqua:
		pool.Aqua += delta
	case AetherElementIgnus:
		pool.Ignus += delta
	case AetherElementLuna:
		pool.Luna += delta
	case AetherElementSilva:
		pool.Silva += delta
	case AetherElementSolis:
		pool.Solis += delta
	case AetherElementTerra:
		pool.Terra += delta
	case AetherElementVoid:
		pool.Void += delta
	case AetherElementNonElemental:
		pool.NonElemental += delta
	}
}
