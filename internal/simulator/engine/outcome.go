package engine

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func finishMatch(state *model.MatchState, result model.MatchResult) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("game in state %q, must be in %q", state.MatchStatus, model.StatusInProgress)
	}
	switch result.Reason {
	case model.EndReasonDeckOut, model.EndReasonZeroOrbs, model.EndReasonSimultaneousLoss:
	default:
		return fmt.Errorf("unsupported match-end reason %q", result.Reason)
	}
	if result.IsDraw {
		if result.Reason != model.EndReasonSimultaneousLoss {
			return fmt.Errorf("draw result must use simultaneous-loss reason")
		}
		if result.Winner != "" || result.Loser != "" {
			return fmt.Errorf("draw result cannot name a winner or loser")
		}
	} else {
		if result.Reason == model.EndReasonSimultaneousLoss {
			return fmt.Errorf("simultaneous-loss reason must be a draw")
		}
		if result.Winner == "" || result.Loser == "" || result.Winner == result.Loser {
			return fmt.Errorf("winner and loser must be distinct match players")
		}
		foundWinner, foundLoser := false, false
		for _, player := range state.Players {
			if player.ID == result.Winner {
				foundWinner = true
			}
			if player.ID == result.Loser {
				foundLoser = true
			}
		}
		if !foundWinner || !foundLoser {
			return fmt.Errorf("winner and loser must be players in this match")
		}
	}
	state.MatchStatus = model.StatusFinished
	state.Result = result
	state.PrioritySequenceOpen = false
	state.PriorityHolder = ""
	state.PassCount = 0
	return nil
}
