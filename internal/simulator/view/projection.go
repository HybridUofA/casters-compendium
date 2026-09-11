package view

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ProjectMatch(state model.MatchState, viewerID model.PlayerID) (MatchView, error) {
	if viewerID != state.Players[0].ID && viewerID != state.Players[1].ID {
		return MatchView{}, fmt.Errorf("viewer ID %q is not in current player IDs", viewerID)
	}
	projection := MatchView{
		ViewerID:    viewerID,
		MatchStatus: state.MatchStatus,
		Revision:    state.Revision,
		Turn:        state.Turn,
	}
	for index, player := range state.Players {
		playerView := PlayerView{
			ID:        player.ID,
			DeckCount: len(player.Deck),
			Aether:    player.Aether,
		}
		handView, err := projectHand(state, player, viewerID)
		if err != nil {
			return MatchView{}, fmt.Errorf("error displaying %q hand: %w", player.ID, err)
		}
		orbView, err := projectOrbs(state, player)
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

func projectOrbs(state model.MatchState, player model.PlayerState) ([]CardView, error) {
	projection := make([]CardView, 0, len(player.Orbs))
	for _, ID := range player.Orbs {
		_, exists := state.CardInstances[ID]
		if !exists {
			return nil, fmt.Errorf("card %q not in card instances", ID)
		}
		cardView := CardView{
			MatchID:  "",
			CardID:   "",
			Face:     model.CardFaceDown,
			ShowFace: false,
		}
		projection = append(projection, cardView)
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
		case instance.Face == model.CardFaceDown && player.ID == viewerID:
			cardView.MatchID = instance.MatchID
			cardView.CardID = instance.CardID
			cardView.ShowFace = false
		case instance.Face == model.CardFaceDown:
			cardView.ShowFace = false
		default:
			return nil, fmt.Errorf("%s card %q has invalid face state %q", zoneName, instance.CardID, instance.Face)
		}
		projection = append(projection, cardView)
	}
	return projection, nil
}
