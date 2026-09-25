package engine

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

func resolveServantCardPlay(state *model.MatchState, link model.ChaseLink, instance model.CardInstance) error {
	controllerIndex := -1
	for index, player := range state.Players {
		if player.ID == link.Controller {
			controllerIndex = index
			break
		}
	}
	if controllerIndex == -1 {
		return fmt.Errorf("controlling player %q not found", link.Controller)
	}
	servantZoneIndex := -1
	for index, card := range state.Players[controllerIndex].ServantZone {
		if card == link.SourceCardID {
			servantZoneIndex = index
		}
	}
	if servantZoneIndex != -1 {
		return fmt.Errorf("card %q is already in the servant zone", link.SourceCardID)
	}
	if link.EntryOrientation != model.OrientationRecovered &&
		link.EntryOrientation != model.OrientationReversed {
		return fmt.Errorf(
			"Servant entry orientation must be Recovered or Reversed, is %q",
			link.EntryOrientation,
		)
	}
	instance.Face = model.CardFaceUp
	instance.Orientation = link.EntryOrientation
	instance.Controller = link.Controller
	state.Players[controllerIndex].ServantZone = append(state.Players[controllerIndex].ServantZone, link.SourceCardID)
	state.CardInstances[link.SourceCardID] = instance
	return nil
}

func resolveConjureCardPlay(state *model.MatchState, link model.ChaseLink, instance model.CardInstance) error {
	controllerIndex := -1
	for index, player := range state.Players {
		if player.ID == link.Controller {
			controllerIndex = index
			break
		}
	}
	if controllerIndex == -1 {
		return fmt.Errorf("controlling player %q not found", link.Controller)
	}
	for _, cardID := range state.Players[controllerIndex].Graveyard {
		if cardID == link.SourceCardID {
			return fmt.Errorf("card %q is already in the graveyard", link.SourceCardID)
		}
	}

	instance.Face = model.CardFaceUp
	instance.Controller = link.Controller
	state.Players[controllerIndex].Graveyard = append(
		state.Players[controllerIndex].Graveyard,
		link.SourceCardID,
	)
	state.CardInstances[link.SourceCardID] = instance
	return nil
}

func resolveBarrierCardPlay(state *model.MatchState, link model.ChaseLink, instance model.CardInstance) error {
	controllerIndex := -1
	for index, player := range state.Players {
		if player.ID == link.Controller {
			controllerIndex = index
			break
		}
	}
	if controllerIndex == -1 {
		return fmt.Errorf("controlling player %q not found", link.Controller)
	}
	for _, cardID := range state.Players[controllerIndex].ServantZone {
		if cardID == link.SourceCardID {
			return fmt.Errorf("card %q is already in the persistent field zone", link.SourceCardID)
		}
	}

	instance.Face = model.CardFaceUp
	instance.Orientation = model.OrientationRecovered
	instance.Controller = link.Controller
	state.Players[controllerIndex].ServantZone = append(
		state.Players[controllerIndex].ServantZone,
		link.SourceCardID,
	)
	state.CardInstances[link.SourceCardID] = instance
	return nil
}

func resolveTopChaseLink(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID) error {
	err := rules.ValidateResolveTopChaseLink(state, actingPlayerID)
	if err != nil {
		return fmt.Errorf("error resolving top chase link: %w", err)
	}
	activePlayerExists := false
	for _, player := range state.Players {
		if player.ID == state.Turn.ActivePlayer {
			activePlayerExists = true
			break
		}
	}

	if !activePlayerExists {
		return fmt.Errorf("active player %q was not found", state.Turn.ActivePlayer)
	}
	topLinkIndex := len(state.ChaseLinks) - 1
	topLink := state.ChaseLinks[topLinkIndex]
	switch topLink.Kind {
	case model.ChaseLinkCardPlay:
		instance, exists := state.CardInstances[topLink.SourceCardID]
		if !exists {
			return fmt.Errorf("card %q was not found in card instances", topLink.SourceCardID)
		}
		definition, err := rules.ResolveCardDefinition(catalog, instance.CardID)
		if err != nil {
			return fmt.Errorf("error resolving definition of card %q: %w", instance.CardID, err)
		}
		switch strings.ToLower(strings.TrimSpace(definition.Type)) {
		case "servant":
			if err := resolveServantCardPlay(state, topLink, instance); err != nil {
				return fmt.Errorf("error resolving Servant card play: %w", err)
			}
		case "conjure":
			// Card-specific Conjure effects are handled manually in the alpha;
			// resolving the card play still sends the card to its Graveyard.
			if err := resolveConjureCardPlay(state, topLink, instance); err != nil {
				return fmt.Errorf("error resolving Conjure card play: %w", err)
			}
		case "barrier":
			// Card-specific Barrier effects are handled manually in the alpha;
			// the persistent card itself enters the shared field zone.
			if err := resolveBarrierCardPlay(state, topLink, instance); err != nil {
				return fmt.Errorf("error resolving Barrier card play: %w", err)
			}
		default:
			return fmt.Errorf("unsupported card-play type %q", definition.Type)
		}
	case model.ChaseLinkActivatedAbility:
		return fmt.Errorf("activated ability resolution is not currently implemented")
	case model.ChaseLinkTriggeredAbility:
		return fmt.Errorf("triggered ability resolution is not currently implemented")
	default:
		return fmt.Errorf("unsupported Chase link kind %q", topLink.Kind)
	}
	state.ChaseLinks = state.ChaseLinks[:topLinkIndex]
	state.PassCount = 0
	state.PriorityHolder = state.Turn.ActivePlayer
	return nil
}

func AddChaseLink(state *model.MatchState, actingPlayerID model.PlayerID, sourceMatchCardID model.MatchCardID, kind model.ChaseLinkKind, expectedRevision model.Revision) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("state expected to be %d, is %d", expectedRevision, state.Revision)
	}
	err := rules.ValidateAddChaseLink(state, actingPlayerID, sourceMatchCardID, kind)
	if err != nil {
		return fmt.Errorf("error occurred during chase link add: %w", err)
	}
	link := model.ChaseLink{
		ID:           state.NextLinkID,
		Controller:   actingPlayerID,
		SourceCardID: sourceMatchCardID,
		Kind:         kind,
	}
	var nextPriorityHolder model.PlayerID
	for _, player := range state.Players {
		if player.ID != actingPlayerID {
			nextPriorityHolder = player.ID
			break
		}
	}
	if nextPriorityHolder == "" {
		return fmt.Errorf("opposing player was not found")
	}
	state.ChaseLinks = append(state.ChaseLinks, link)
	state.PassCount = 0
	state.PriorityHolder = nextPriorityHolder
	state.NextLinkID++
	state.Revision++
	return nil
}

func PassPriority(state *model.MatchState, catalog rules.CardCatalog, actingPlayerID model.PlayerID, expectedRevision model.Revision) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if expectedRevision != state.Revision {
		return fmt.Errorf("state expected to be %d, is %d", expectedRevision, state.Revision)
	}
	err := rules.ValidatePassPriority(state, actingPlayerID)
	if err != nil {
		return fmt.Errorf("error occurred during passing priority: %w", err)
	}
	var nextPriorityHolder model.PlayerID
	for _, player := range state.Players {
		if player.ID != actingPlayerID {
			nextPriorityHolder = player.ID
			break
		}
	}
	if nextPriorityHolder == "" {
		return fmt.Errorf("opposing player was not found")
	}
	if state.PassCount == 0 {
		state.PassCount = 1
		state.PriorityHolder = nextPriorityHolder
		state.Revision++
		return nil
	}
	if state.PassCount == 1 {
		if len(state.ChaseLinks) > 0 {
			err := resolveTopChaseLink(state, catalog, actingPlayerID)
			if err != nil {
				return fmt.Errorf("error during resolution: %w", err)
			}
			state.Revision++
			return nil
		}
		if len(state.ChaseLinks) == 0 {
			// close priority window
			return fmt.Errorf("second pass not currently implemented")
		}
	}
	return nil
}
