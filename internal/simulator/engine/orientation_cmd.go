package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// SetFieldOrientation manually sets Recovered / Rested / Reversed on a field
// card the acting player controls. Same-zone MoveCard cannot change orientation
// in place, so this is the Cockatrice-style aid for that common case.
// catalog may be nil; when present, World of Rage blocks manual Switch
// (Recovered↔Reversed) on affected enemy servants.
func SetFieldOrientation(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	orientation model.CardOrientation,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if strings.TrimSpace(string(cardID)) == "" {
		return fmt.Errorf("card ID cannot be empty")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("only the controller may change orientation of %q", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	switch location.Zone {
	case model.ZoneServant:
		if cardIsBarrier(catalog, instance.CardID) {
			if orientation != model.OrientationRecovered && orientation != model.OrientationRested {
				return fmt.Errorf("barrier orientation must be Recovered or Rested")
			}
			break
		}
		if orientation != model.OrientationRecovered &&
			orientation != model.OrientationRested &&
			orientation != model.OrientationReversed {
			return fmt.Errorf("servant orientation must be Recovered, Rested, or Reversed")
		}
	case model.ZoneCaster:
		if orientation != model.OrientationRecovered && orientation != model.OrientationRested {
			return fmt.Errorf("caster orientation must be Recovered or Rested")
		}
	default:
		return fmt.Errorf("orientation can only be set on Servant or Caster zone cards")
	}
	if instance.Orientation == orientation {
		return fmt.Errorf("card is already %s", orientation)
	}
	// World of Rage: enemy servants cannot be switched except by spell/ability.
	// Manual Recovered↔Reversed counts as a Switch; Rest is still allowed.
	if location.Zone == model.ZoneServant && catalog != nil {
		playerIndex, _ := findPlayerIndex(state, actingPlayerID)
		if worldOfRageAffects(state, catalog, playerIndex) {
			switching := (instance.Orientation == model.OrientationRecovered && orientation == model.OrientationReversed) ||
				(instance.Orientation == model.OrientationReversed && orientation == model.OrientationRecovered)
			if switching {
				return fmt.Errorf("World of Rage: enemy servants cannot be switched except by spell or ability")
			}
		}
	}
	if err := ops.SetOrientation(state, cardID, orientation); err != nil {
		return err
	}
	clearAttackIfInterrupted(state)
	state.Revision++
	return nil
}
