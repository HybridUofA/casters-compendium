package model

import (
	"fmt"
)

func (state MatchState) ValidateInitialState() error {
	if state.Players[0].ID == "" || state.Players[1].ID == "" || state.Players[0].ID == state.Players[1].ID {
		return fmt.Errorf("player ID cannot be empty or the same")
	}
	if state.FirstPlayer != state.Players[0].ID && state.FirstPlayer != state.Players[1].ID {
		return fmt.Errorf("no first player set")
	}
	if state.Turn.ActivePlayer != state.Players[0].ID && state.Turn.ActivePlayer != state.Players[1].ID {
		return fmt.Errorf("no active player set")
	}
	if state.Turn.ActivePlayer != state.FirstPlayer {
		return fmt.Errorf("%q is the active player, but %q is the player going first", state.Turn.ActivePlayer, state.FirstPlayer)
	}
	seen := make(map[MatchCardID]string)
	for _, player := range state.Players {
		for _, cardID := range player.Deck {
			location := fmt.Sprintf("player %s deck", player.ID)
			instance, exists := state.CardInstances[cardID]
			if !exists {
				return fmt.Errorf("%s references unknown card %q", location, cardID)
			}
			if instance.Owner != player.ID {
				return fmt.Errorf("card %q in %s is owned by %q", cardID, location, instance.Owner)
			}
			if previousLocation, alreadySeen := seen[cardID]; alreadySeen {
				return fmt.Errorf("card %q appears in both %s and %s", cardID, previousLocation, location)
			}
			seen[cardID] = location
		}
		for _, cardID := range player.Hand {
			location := fmt.Sprintf("player %s hand", player.ID)
			instance, exists := state.CardInstances[cardID]
			if !exists {
				return fmt.Errorf("%s references unknown card %q", location, cardID)
			}
			if instance.Owner != player.ID {
				return fmt.Errorf("card %q in %s is owned by %q", cardID, location, instance.Owner)
			}
			if previousLocation, alreadySeen := seen[cardID]; alreadySeen {
				return fmt.Errorf("card %q appears in both %s and %s", cardID, previousLocation, location)
			}
			seen[cardID] = location
		}
		for _, cardID := range player.Orbs {
			location := fmt.Sprintf("player %s orbs", player.ID)
			instance, exists := state.CardInstances[cardID]
			if !exists {
				return fmt.Errorf("%s references unknown card %q", location, cardID)
			}
			if instance.Owner != player.ID {
				return fmt.Errorf("card %q in %s is owned by %q", cardID, location, instance.Owner)
			}
			if previousLocation, alreadySeen := seen[cardID]; alreadySeen {
				return fmt.Errorf("card %q appears in both %s and %s", cardID, previousLocation, location)
			}
			seen[cardID] = location
		}

		if len(player.CasterZone) != 1 {
			return fmt.Errorf("expected 1 card in caster zone, got %d", len(player.CasterZone))
		}
		for _, cardID := range player.CasterZone {
			location := fmt.Sprintf("player %s caster zone", player.ID)
			instance, exists := state.CardInstances[cardID]
			if !exists {
				return fmt.Errorf("%s references unknown card %q", location, cardID)
			}
			if instance.Owner != player.ID {
				return fmt.Errorf("card %q in %s is owned by %q", cardID, location, instance.Owner)
			}
			if previousLocation, alreadySeen := seen[cardID]; alreadySeen {
				return fmt.Errorf("card %q appears in both %s and %s", cardID, previousLocation, location)
			}
			seen[cardID] = location
			if instance.CardID != CasterTokenCardID || instance.CardCategory != CategoryTokenCard || instance.Face != CardFaceUp || instance.Orientation != OrientationRecovered {
				return fmt.Errorf("caster token in unexpected state")
			}
			if player.ID != state.CardInstances[cardID].Controller {
				return fmt.Errorf("caster token in %q's zone is controlled by %q", instance.Owner, instance.Controller)
			}
		}
		if len(player.Orbs) != 7 {
			return fmt.Errorf("%q must have 7 orbs, but has %d", player.ID, len(player.Orbs))
		}
		if len(player.Hand) != 7 {
			return fmt.Errorf("%q must have 7 cards in hand, but has %d", player.ID, len(player.Hand))
		}
		if total := len(player.Deck) + len(player.Hand) + len(player.Orbs); total != 50 {
			return fmt.Errorf("total number of cards is not 50, it is %d", total)
		}
	}
	for matchID := range state.CardInstances {
		if _, wasSeen := seen[matchID]; !wasSeen {
			return fmt.Errorf("card instance %q is not assigned to any player zone", matchID)
		}
	}
	for matchID, instance := range state.CardInstances {
		if matchID == "" {
			return fmt.Errorf("match ID cannot be empty")
		}
		if instance.CardID == "" {
			return fmt.Errorf("card ID cannot be empty")
		}
		if instance.MatchID != matchID {
			return fmt.Errorf("%q not matching %q", instance.MatchID, matchID)
		}
		if instance.Owner != state.Players[0].ID && instance.Owner != state.Players[1].ID {
			return fmt.Errorf("card instance %q has unknown owner %q", matchID, instance.Owner)
		}
		if instance.Controller != instance.Owner {
			return fmt.Errorf("card instance %q has controller %q, but initial owner is %q", matchID, instance.Controller, instance.Owner)
		}
	}
	if state.MatchStatus != StatusInProgress {
		return fmt.Errorf("match in %q status, but must be %q", state.MatchStatus, StatusInProgress)
	}
	if state.Turn.Number != 1 {
		return fmt.Errorf("turn count is %d, must be 1", state.Turn.Number)
	}
	if state.Turn.Phase != PhaseCall {
		return fmt.Errorf("current phase is %q, must be %q", state.Turn.Phase, PhaseCall)
	}
	return nil
}
