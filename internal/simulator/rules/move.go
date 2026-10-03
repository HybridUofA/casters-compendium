package rules

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateMoveCard(state *model.MatchState, catalog CardCatalog, actingPlayerID model.PlayerID, command model.MoveCardCommand) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match must be in progress")
	}
	actingPlayerIndex := -1
	for index, player := range state.Players {
		if player.ID == actingPlayerID {
			actingPlayerIndex = index
			break
		}
	}
	if actingPlayerIndex == -1 {
		return fmt.Errorf("player %q not found", actingPlayerID)
	}
	location, err := model.FindCardLocation(state, command.CardID)
	if err != nil {
		return fmt.Errorf("error finding card: %w", err)
	}
	destinationPlayerIndex := -1
	for index, player := range state.Players {
		if player.ID == command.DestinationPlayerID {
			destinationPlayerIndex = index
			break
		}
	}
	if destinationPlayerIndex == -1 {
		return fmt.Errorf("player %q not found", command.DestinationPlayerID)
	}
	if location.PlayerIndex == destinationPlayerIndex && location.Zone == command.DestinationZone {
		return fmt.Errorf("card cannot be moved to the same zone it is already in")
	}
	instance, exists := state.CardInstances[command.CardID]
	if !exists {
		return fmt.Errorf("card %q does not exist in card instances", command.CardID)
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("controller of %q %q is not the acting player %q", instance.CardID, instance.Controller, actingPlayerID)
	}
	if instance.CardCategory == model.CategoryTokenCard {
		return fmt.Errorf("token movement not yet supported")
	}
	if len(instance.Stock) > 0 {
		return fmt.Errorf("cards with stock movement not yet supported")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if command.DestinationFace != model.CardFaceUp && command.DestinationFace != model.CardFaceDown {
		return fmt.Errorf("card must either be face-up or face-down")
	}
	if command.DestinationZone == model.ZoneDeck {
		if command.Placement != model.DeckPlacementBottom && command.Placement != model.DeckPlacementTop {
			return fmt.Errorf("card being added to deck must either be placed on top or on the bottom of the deck")
		}
	} else if command.Placement != "" {
		return fmt.Errorf("other zones cannot have placement decisions")
	}
	switch command.DestinationZone {
	case model.ZoneCaster:
		if command.EntryOrientation != model.OrientationRecovered && command.EntryOrientation != model.OrientationRested {
			return fmt.Errorf("caster entering in an invalid state: %q", command.EntryOrientation)
		}
		if command.DestinationFace == model.CardFaceUp {
			if !strings.EqualFold(strings.TrimSpace(definition.Type), "Caster") {
				return fmt.Errorf("face-up cards being put into the caster zone must be casters")
			}
		}
	case model.ZoneServant:
		if command.DestinationFace != model.CardFaceUp {
			return fmt.Errorf("cards put into servant zone must be face-up")
		}
		switch strings.ToLower(strings.TrimSpace(definition.Type)) {
		case "servant":
			if command.EntryOrientation != model.OrientationRecovered && command.EntryOrientation != model.OrientationRested && command.EntryOrientation != model.OrientationReversed {
				return fmt.Errorf("servant entering in an invalid state: %q", command.EntryOrientation)
			}
		case "barrier":
			if command.EntryOrientation != model.OrientationRecovered && command.EntryOrientation != model.OrientationRested {
				return fmt.Errorf("barrier entering in an invalid state: %q", command.EntryOrientation)
			}
		default:
			return fmt.Errorf("unsupported or invalid card type for servant zone: %q", definition.Type)
		}
	case model.ZoneHand, model.ZoneDeck, model.ZoneGraveyard, model.ZoneExile, model.ZoneOrbs:
		if command.EntryOrientation != "" {
			return fmt.Errorf("non-field destinations must not specify an entry orientation")
		}
		if command.DestinationPlayerID != instance.Owner {
			return fmt.Errorf("card being moved to %q's %q, must belong to %q", command.DestinationPlayerID, command.DestinationZone, instance.Owner)
		}
		switch command.DestinationZone {
		case model.ZoneHand, model.ZoneDeck, model.ZoneOrbs:
			if command.DestinationFace != model.CardFaceDown {
				return fmt.Errorf("cards that enter the hand, deck, or orbs must be face-down")
			}
		case model.ZoneGraveyard:
			if command.DestinationFace != model.CardFaceUp {
				return fmt.Errorf("cards that enter the graveyard must be face-up")
			}
		default:
		}
	default:
		return fmt.Errorf("unsupported destination zone %q", command.DestinationZone)
	}
	// This manual command treats a move between corresponding field zones as
	// a control transfer, not an opportunity to flip or recover the card.
	if location.Zone == command.DestinationZone &&
		(location.Zone == model.ZoneCaster || location.Zone == model.ZoneServant) &&
		location.PlayerIndex != destinationPlayerIndex {
		if command.DestinationFace != instance.Face || command.EntryOrientation != instance.Orientation {
			return fmt.Errorf("control transfers must preserve the card's face and orientation")
		}
	}
	return nil
}
