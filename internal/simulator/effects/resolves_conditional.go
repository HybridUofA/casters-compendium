package effects

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

const (
	PendingHildeDeckBottom = "fx:hilde_deck_bottom"
	PendingRPSChoice       = "fx:rps_choice"
)

// --- Conditional Rest draws ---

func resolveRestDrawIfTwoAlliedEntered(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.ServantsEntered[idx] >= 2
	})
}

func resolveRestDrawIfBattleDestroy(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.BattleDestroysDealt[idx] >= 1
	})
}

func resolveRestDrawIfAlliedDestroyedNonBattle(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.AlliedDestroyedNonBattle[idx] >= 1
	})
}

func resolveRestDrawIfAlliedDestroyedInBattle(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.AlliedDestroyedInBattle[idx] >= 1
	})
}

func resolveRestDrawIfInsectDestroyedOrBanished(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.AlliedInsectDestroyedOrBanished[idx] >= 1
	})
}

func resolveRestDrawIfCorruptedOrb(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.AlliedCorruptedOrb[idx] >= 1
	})
}

func resolveRestDrawIfEnemyDiscarded(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.EnemyDiscarded[idx] >= 1
	})
}

func resolveRestDrawIfEnemyReturned(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.EnemyReturnedToHand[idx] >= 1
	})
}

func resolveRestDrawIfEnemyEnter(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.EnemyEnterAbility[idx] >= 1
	})
}

func resolveRestDrawIfBreakPlayed(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.BreakPlayed[idx] >= 1
	})
}

func resolveRestDrawIfNoAlliedPlayed(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.AnyCardPlayed[idx] == 0
	})
}

func resolveRestDrawIfNoNonCasterPlayed(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.NonCasterPlayed[idx] == 0
	})
}

func resolveRestDrawIfConjureDraw(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.ConjureEffectDraws[idx] >= 1
	})
}

func resolveRestDrawIfEnemyAttackedTwice(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.EnemyAttacks[idx] >= 2
	})
}

func resolveRestDrawIfSpellDestroyedEnemy(ctx Context) error {
	return drawIf(ctx, func(flags model.TurnFlags, idx int) bool {
		return flags.SpellDestroyedEnemy[idx] >= 1
	})
}

func drawIf(ctx Context, cond func(model.TurnFlags, int) bool) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	if !cond(ctx.State.TurnFlags, idx) {
		return nil
	}
	return ops.Draw(ctx.State, ctx.Controller, 1)
}

// --- Hilde Willow ---

func resolveHildeWillowRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	controllerIdx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	enemyIdx := 1 - controllerIdx
	enemyID := ctx.State.Players[enemyIdx].ID
	deck := ctx.State.Players[enemyIdx].Deck
	if len(deck) == 0 {
		return fmt.Errorf("enemy deck is empty")
	}
	top := deck[0]
	model.MarkCardKnown(ctx.State, ctx.Controller, top)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:     ctx.Controller,
		Kind:         PendingHildeDeckBottom,
		Prompt:       "Enemy deck top revealed. Leave it on top, or put it on the bottom?",
		SourceCardID: ctx.Source,
		Payload:      string(enemyID),
		Options: []model.PendingChoiceOption{
			{ID: "leave", Label: "Leave on top"},
			{ID: "bottom", Label: "Put on bottom"},
		},
	})
}

// --- Rock-paper-scissors ---

func resolveGoblinMoneylenderEnter(ctx Context) error {
	return offerRPS(ctx, "enter")
}

func resolveGoblinMoneylenderLastWords(ctx Context) error {
	return offerRPS(ctx, "last_words")
}

func offerRPS(ctx Context, mode string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	controllerIdx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	enemyID := ctx.State.Players[1-controllerIdx].ID
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:     ctx.Controller,
		Kind:         PendingRPSChoice,
		Prompt:       "Choose Rock, Paper, or Scissors",
		SourceCardID: ctx.Source,
		Payload:      fmt.Sprintf("%s|%s|", mode, enemyID),
		Options: []model.PendingChoiceOption{
			{ID: "rock", Label: "Rock"},
			{ID: "paper", Label: "Paper"},
			{ID: "scissors", Label: "Scissors"},
		},
	})
}

func continueHildeDeckBottom(pending model.PendingChoice, optionID string, ctx Context) error {
	if optionID == "leave" || optionID == "" {
		return nil
	}
	if optionID != "bottom" {
		return fmt.Errorf("unknown Hilde option %q", optionID)
	}
	enemyID := model.PlayerID(pending.Payload)
	if enemyID == "" {
		return fmt.Errorf("hilde pending missing enemy payload")
	}
	return ops.PutDeckTopOnBottom(ctx.State, enemyID)
}

func continueRPSChoice(pending model.PendingChoice, optionID string, ctx Context) error {
	parts := strings.Split(pending.Payload, "|")
	if len(parts) < 2 {
		return fmt.Errorf("rps payload malformed")
	}
	mode := parts[0]
	enemyID := model.PlayerID(parts[1])
	firstChoice := ""
	if len(parts) >= 3 {
		firstChoice = parts[2]
	}
	switch optionID {
	case "rock", "paper", "scissors":
	default:
		return fmt.Errorf("invalid rps choice %q", optionID)
	}
	if firstChoice == "" {
		// First player chose; ask the opponent.
		return offerOrSkip(ctx, model.PendingChoice{
			PlayerID:     enemyID,
			Kind:         PendingRPSChoice,
			Prompt:       "Choose Rock, Paper, or Scissors",
			SourceCardID: pending.SourceCardID,
			Payload:      fmt.Sprintf("%s|%s|%s", mode, pending.PlayerID, optionID),
			Options: []model.PendingChoiceOption{
				{ID: "rock", Label: "Rock"},
				{ID: "paper", Label: "Paper"},
				{ID: "scissors", Label: "Scissors"},
			},
		})
	}
	// Payload after first pick: mode|firstPlayer|firstChoice; PlayerID is the second chooser.
	originalFirst := model.PlayerID(parts[1])
	secondPlayer := pending.PlayerID
	winner := rpsWinner(firstChoice, optionID, originalFirst, secondPlayer)
	switch mode {
	case "enter":
		if winner == "" {
			return nil // tie: nobody draws
		}
		return ops.Draw(ctx.State, winner, 2)
	case "last_words":
		loser := secondPlayer
		if winner == secondPlayer {
			loser = originalFirst
		}
		if winner == "" {
			return nil
		}
		return offerDiscardChain(ctxWithController(ctx, loser), 3)
	case "necro":
		if winner == "" {
			return nil
		}
		if winner == originalFirst {
			return resolveSearchAnyCardToHand(ctxWithController(ctx, originalFirst))
		}
		return ops.AddAether(ctx.State, originalFirst, "Void", 3)
	default:
		return fmt.Errorf("unknown rps mode %q", mode)
	}
}

func ctxWithController(ctx Context, controller model.PlayerID) Context {
	out := ctx
	out.Controller = controller
	return out
}

func rpsWinner(first, second string, firstPlayer, secondPlayer model.PlayerID) model.PlayerID {
	if first == second {
		return ""
	}
	wins := map[string]string{
		"rock":     "scissors",
		"paper":    "rock",
		"scissors": "paper",
	}
	if wins[first] == second {
		return firstPlayer
	}
	return secondPlayer
}

// --- Continuous until-EOT grants ---

func resolveJustAShortRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	ctx.State.ContinuousUntilEOT.AlliedServantsCannotBeAttacked[idx] = true
	return nil
}

func resolveCharylThomasRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	ally := len(ctx.State.Players[idx].ServantZone)
	enemy := len(ctx.State.Players[1-idx].ServantZone)
	if ally >= enemy {
		return nil
	}
	ctx.State.ContinuousUntilEOT.RestedAlliedCannotBeAttacked[idx] = true
	return nil
}
