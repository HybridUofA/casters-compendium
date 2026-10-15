package view

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// IsMatchPlayer reports whether viewerID occupies a seated player slot.
func IsMatchPlayer(state model.MatchState, viewerID model.PlayerID) bool {
	return viewerID == state.Players[0].ID || viewerID == state.Players[1].ID
}

func projectPendingBreak(pending model.PendingBreak, viewerID model.PlayerID) model.PendingBreak {
	if pending.PlayerID == "" {
		return model.PendingBreak{}
	}
	if pending.PlayerID != viewerID {
		// Peers only learn that a Break decision is outstanding.
		return model.PendingBreak{PlayerID: pending.PlayerID}
	}
	out := model.PendingBreak{
		PlayerID: pending.PlayerID,
		CardIDs:  append([]model.MatchCardID(nil), pending.CardIDs...),
	}
	return out
}

func projectPendingChoice(pending model.PendingChoice, viewerID model.PlayerID) model.PendingChoice {
	if pending.PlayerID == "" {
		return model.PendingChoice{}
	}
	if pending.PlayerID != viewerID {
		// Peers only learn that a decision is outstanding.
		return model.PendingChoice{PlayerID: pending.PlayerID}
	}
	out := model.PendingChoice{
		PlayerID:     pending.PlayerID,
		Kind:         pending.Kind,
		Prompt:       pending.Prompt,
		AllowDecline: pending.AllowDecline,
	}
	if len(pending.Options) > 0 {
		out.Options = append([]model.PendingChoiceOption(nil), pending.Options...)
	}
	if len(pending.SelectableCardIDs) > 0 {
		out.SelectableCardIDs = append([]model.MatchCardID(nil), pending.SelectableCardIDs...)
	}
	return out
}

// IsSpectatorViewer reports whether viewerID is a recognized spectator id.
func IsSpectatorViewer(viewerID model.PlayerID) bool {
	return strings.HasPrefix(string(viewerID), "spectator-")
}

func ProjectMatch(state model.MatchState, viewerID model.PlayerID) (MatchView, error) {
	isPlayer := IsMatchPlayer(state, viewerID)
	if !isPlayer && !IsSpectatorViewer(viewerID) {
		return MatchView{}, fmt.Errorf("viewer ID %q is not in current player IDs", viewerID)
	}
	projection := MatchView{
		ViewerID:             viewerID,
		MatchStatus:          state.MatchStatus,
		Revision:             state.Revision,
		Turn:                 state.Turn,
		FirstPlayer:          state.FirstPlayer,
		PriorityHolder:       state.PriorityHolder,
		PassCount:            state.PassCount,
		ChaseLinkCount:       len(state.ChaseLinks),
		ChaseLinks:           projectChaseLinks(state),
		PrioritySequenceOpen: state.PrioritySequenceOpen,
		Attack:               state.Attack,
		PendingDraw:          state.PendingDraw,
		PendingBreak:         projectPendingBreak(state.PendingBreak, viewerID),
		PendingChoice:        projectPendingChoice(state.PendingChoice, viewerID),
		Result:               state.Result,
		Spectator:            !isPlayer,
	}
	for index, player := range state.Players {
		playerView := PlayerView{
			ID:        player.ID,
			DeckCount: len(player.Deck),
			Aether:    player.Aether,
		}
		deckView, err := projectDeck(state, player, viewerID)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q deck: %w", player.ID, err)
		}
		handView, err := projectHand(state, player, viewerID)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q hand: %w", player.ID, err)
		}
		orbView, err := projectOrbs(state, player, viewerID)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q orbs: %w", player.ID, err)
		}
		casterZoneView, err := projectFieldZone(state, player, viewerID, "caster zone", player.CasterZone)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q caster zone: %w", player.ID, err)
		}
		servantZoneView, err := projectFieldZone(state, player, viewerID, "servant zone", player.ServantZone)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q servant zone: %w", player.ID, err)
		}
		graveyardView, err := projectFieldZone(state, player, viewerID, "graveyard", player.Graveyard)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q graveyard: %w", player.ID, err)
		}
		exileView, err := projectFieldZone(state, player, viewerID, "removed from game zone", player.Exile)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying removed from game zone: %w", err)
		}
		playerView.Deck = deckView
		playerView.Hand = handView
		playerView.Orbs = orbView
		playerView.CasterZone = casterZoneView
		playerView.ServantZone = servantZoneView
		playerView.Graveyard = graveyardView
		playerView.OpeningHandFinalized = player.OpeningHandFinalized
		playerView.Exile = exileView
		projection.Players[index] = playerView
	}
	return projection, nil
}

func projectDeck(state model.MatchState, player model.PlayerState, viewerID model.PlayerID) ([]CardView, error) {
	if player.ID != viewerID {
		return nil, nil
	}
	projection := make([]CardView, 0, len(player.Deck))
	for _, ID := range player.Deck {
		instance, exists := state.CardInstances[ID]
		if !exists {
			return nil, fmt.Errorf("card %q not in card instances", ID)
		}
		projection = append(projection, CardView{
			MatchID:  ID,
			CardID:   instance.CardID,
			Face:     model.CardFaceDown,
			ShowFace: false,
			Owner:    instance.Owner,
			HasStock: len(instance.Stock) > 0,
		})
	}
	return projection, nil
}

func projectHand(state model.MatchState, player model.PlayerState, viewerID model.PlayerID) ([]CardView, error) {
	projection := make([]CardView, 0, len(player.Hand))
	for _, ID := range player.Hand {
		instance, exists := state.CardInstances[ID]
		if !exists {
			return nil, fmt.Errorf("card %q not in card instances", ID)
		}
		if player.ID == viewerID {
			cardView := CardView{
				MatchID:  ID,
				CardID:   instance.CardID,
				ShowFace: true,
				Owner:    instance.Owner,
				HasStock: len(instance.Stock) > 0,
			}
			projection = append(projection, cardView)
			continue
		}
		cardView := CardView{
			MatchID:  "",
			CardID:   "",
			ShowFace: false,
		}
		projection = append(projection, cardView)
	}
	return projection, nil
}

func projectOrbs(state model.MatchState, player model.PlayerState, viewerID model.PlayerID) ([]CardView, error) {
	projection := make([]CardView, 0, len(player.Orbs))
	for _, ID := range player.Orbs {
		instance, exists := state.CardInstances[ID]
		if !exists {
			return nil, fmt.Errorf("card %q not in card instances", ID)
		}
		// MatchID is always projected so UI can target Orbs (Corrupt/Banish).
		// Card identity stays hidden unless this viewer already knows them.
		view := CardView{
			MatchID:  ID,
			Face:     model.CardFaceDown,
			ShowFace: false,
			Owner:    instance.Owner,
		}
		if model.ViewerKnowsCard(&state, viewerID, ID) {
			view.CardID = instance.CardID
		}
		projection = append(projection, view)
	}
	return projection, nil
}

func projectFieldZone(
	state model.MatchState,
	player model.PlayerState,
	viewerID model.PlayerID,
	zoneName string,
	cardIDs []model.MatchCardID,
) ([]CardView, error) {
	projection := make([]CardView, 0, len(cardIDs))
	for _, ID := range cardIDs {
		instance, exists := state.CardInstances[ID]
		if !exists {
			return nil, fmt.Errorf("%s card %q not in card instances", zoneName, ID)
		}
		cardView := CardView{
			Face:        instance.Face,
			Orientation: instance.Orientation,
		}
		switch {
		case instance.Face == model.CardFaceUp:
			cardView.MatchID = instance.MatchID
			cardView.CardID = instance.CardID
			cardView.ShowFace = true
			cardView.Owner = instance.Owner
			cardView.HasStock = len(instance.Stock) > 0
			cardView.GrantedDoubleCorrupt = instance.GrantedDoubleCorrupt
			cardView.AttackModifier = instance.AttackModifier
			cardView.DefenseModifier = instance.DefenseModifier
		case instance.Face == model.CardFaceDown && player.ID == viewerID:
			cardView.MatchID = instance.MatchID
			cardView.CardID = instance.CardID
			cardView.ShowFace = false
			cardView.Owner = instance.Owner
			cardView.HasStock = len(instance.Stock) > 0
			cardView.GrantedDoubleCorrupt = instance.GrantedDoubleCorrupt
			cardView.AttackModifier = instance.AttackModifier
			cardView.DefenseModifier = instance.DefenseModifier
		case instance.Face == model.CardFaceDown:
			cardView.ShowFace = false
		default:
			return nil, fmt.Errorf("%s card %q has invalid face state %q", zoneName, instance.CardID, instance.Face)
		}
		projection = append(projection, cardView)
	}
	return projection, nil
}

// projectChaseLinks exposes the Chase stack publicly. Cards on Chase are always
// face-up with known identity (both players must see what is resolving).
func projectChaseLinks(state model.MatchState) []ChaseLinkView {
	if len(state.ChaseLinks) == 0 {
		return nil
	}
	out := make([]ChaseLinkView, 0, len(state.ChaseLinks))
	for _, link := range state.ChaseLinks {
		card := CardView{
			Face:        model.CardFaceUp,
			Orientation: link.EntryOrientation,
			ShowFace:    true,
		}
		if instance, ok := state.CardInstances[link.SourceCardID]; ok {
			card.MatchID = instance.MatchID
			card.CardID = instance.CardID
			card.Owner = instance.Owner
			card.HasStock = len(instance.Stock) > 0
			card.GrantedDoubleCorrupt = instance.GrantedDoubleCorrupt
			card.AttackModifier = instance.AttackModifier
			card.DefenseModifier = instance.DefenseModifier
			if link.EntryOrientation == "" {
				card.Orientation = instance.Orientation
			}
		} else {
			card.MatchID = link.SourceCardID
		}
		out = append(out, ChaseLinkView{
			ID:               link.ID,
			Controller:       link.Controller,
			Kind:             link.Kind,
			EntryOrientation: link.EntryOrientation,
			Card:             card,
		})
	}
	return out
}
