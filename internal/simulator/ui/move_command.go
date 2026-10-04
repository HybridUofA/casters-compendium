package ui

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func buildDefaultMoveCommand(
	card simulatorview.CardView,
	sourceZone model.Zone,
	destinationPlayerID model.PlayerID,
	destinationZone model.Zone,
	definition cards.Card,
	deckPlacement model.DeckPlacement,
) (model.MoveCardCommand, error) {
	if card.MatchID == "" {
		return model.MoveCardCommand{}, fmt.Errorf("card identity is unknown")
	}
	if card.HasStock {
		return model.MoveCardCommand{}, fmt.Errorf("cards with Stock cannot be moved yet")
	}
	if card.CardID == model.CasterTokenCardID {
		return model.MoveCardCommand{}, fmt.Errorf("token movement is not supported")
	}
	if sourceZone == destinationZone {
		// Control transfer uses the same zone name on the other player.
		if card.Owner == destinationPlayerID || destinationPlayerID == "" {
			return model.MoveCardCommand{}, fmt.Errorf("choose a different zone")
		}
	}

	field := destinationZone == model.ZoneCaster || destinationZone == model.ZoneServant
	destination := destinationPlayerID
	if !field && card.Owner != "" {
		destination = card.Owner
	}
	if destination == "" {
		return model.MoveCardCommand{}, fmt.Errorf("destination player is required")
	}

	face := model.CardFaceDown
	orientation := model.CardOrientation("")
	placement := model.DeckPlacement("")
	kind := strings.ToLower(strings.TrimSpace(definition.Type))

	switch destinationZone {
	case model.ZoneHand:
		face = model.CardFaceDown
	case model.ZoneDeck:
		face = model.CardFaceDown
		if deckPlacement == "" {
			deckPlacement = model.DeckPlacementTop
		}
		placement = deckPlacement
	case model.ZoneOrbs:
		face = model.CardFaceDown
	case model.ZoneGraveyard:
		face = model.CardFaceUp
	case model.ZoneExile:
		face = model.CardFaceUp
	case model.ZoneCaster:
		orientation = model.OrientationRecovered
		if kind == "caster" {
			face = model.CardFaceUp
		} else {
			face = model.CardFaceDown
		}
	case model.ZoneServant:
		if kind != "servant" && kind != "barrier" {
			return model.MoveCardCommand{}, fmt.Errorf("%s cannot enter the Servant zone", definition.Type)
		}
		face = model.CardFaceUp
		orientation = model.OrientationRecovered
	default:
		return model.MoveCardCommand{}, fmt.Errorf("unsupported destination zone %q", destinationZone)
	}

	return model.MoveCardCommand{
		CardID:              card.MatchID,
		DestinationPlayerID: destination,
		DestinationZone:     destinationZone,
		DestinationFace:     face,
		EntryOrientation:    orientation,
		Placement:           placement,
	}, nil
}

func cardMovable(card simulatorview.CardView) bool {
	return card.MatchID != "" && !card.HasStock && card.CardID != model.CasterTokenCardID
}
