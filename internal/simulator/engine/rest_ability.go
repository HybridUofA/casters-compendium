package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// ActivateRegisteredRest pays Rest on a recovered field card and runs HookRest
// handlers. Activated-ability Chase resolution is not implemented yet (same as
// Hilde Willow): the ability resolves immediately and the acting player keeps
// priority. Cards without a registered Rest handler are rejected.
func ActivateRegisteredRest(
	state *model.MatchState,
	catalog rules.CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", expectedRevision, state.Revision)
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match is not in progress")
	}
	if !state.PrioritySequenceOpen || state.PriorityHolder != actingPlayerID {
		return fmt.Errorf("only the priority holder may activate a Rest ability")
	}
	if state.PendingBreak.PlayerID != "" ||
		state.PendingDraw.Step != model.PendingDrawIdle ||
		state.PendingChoice.PlayerID != "" {
		return fmt.Errorf("cannot activate a Rest ability while another decision is pending")
	}

	instance, ok := state.CardInstances[cardID]
	if !ok {
		return fmt.Errorf("card %q does not exist", cardID)
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("you do not control %q", cardID)
	}
	location, err := model.FindCardLocation(state, cardID)
	if err != nil {
		return fmt.Errorf("locate card: %w", err)
	}
	if location.Zone != model.ZoneServant && location.Zone != model.ZoneCaster {
		return fmt.Errorf("card %q is not on the field", cardID)
	}
	if instance.Orientation != model.OrientationRecovered {
		return fmt.Errorf("card %q must be recovered to Rest", cardID)
	}

	abilityText := ""
	if catalog != nil {
		if definition, found := catalog.FindByID(string(instance.CardID)); found {
			abilityText = definition.Ability
		}
	}
	grantedVoidRest := instance.GrantedVoidRestATK500UntilEOT
	hasPrintedRest := effects.HasMatch(&effectRegistry, effects.HookRest, instance.CardID, abilityText)
	if !hasPrintedRest && !grantedVoidRest {
		return fmt.Errorf("card %q has no registered Rest ability automation", cardID)
	}

	if err := ops.SetOrientation(state, cardID, model.OrientationRested); err != nil {
		return err
	}
	if grantedVoidRest {
		if err := effects.ResolveGrantedVoidRest(effects.Context{
			State:      state,
			Source:     cardID,
			Controller: actingPlayerID,
			Hook:       effects.HookRest,
			Catalog:    catalog,
			OfferChoice: func(choice model.PendingChoice) error {
				return OfferPendingChoice(state, choice)
			},
		}); err != nil {
			return err
		}
	} else if err := triggerRestEffects(state, catalog, cardID); err != nil {
		return err
	}
	if hasPrintedRest {
		noteIdolNonAetherAbility(state, catalog, actingPlayerID, instance.CardID)
	}
	state.Revision++
	return nil
}
