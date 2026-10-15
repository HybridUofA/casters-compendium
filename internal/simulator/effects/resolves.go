package effects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Pending Kind keys for ContinuePendingChoice.
const (
	PendingDestroyEnemyServant = "fx:destroy_enemy_servant"
	PendingBanishAlliedServant = "fx:banish_allied_servant"
	PendingDiscardOneFromHand  = "fx:discard_one_from_hand"
	PendingDiscardTwoFromHand  = "fx:discard_two_from_hand"
)

// ContinuePendingChoice finishes a targeting / discard prompt opened by a resolve.
// Unknown kinds are ignored (legacy / test prompts).
func ContinuePendingChoice(
	pending model.PendingChoice,
	selected []model.MatchCardID,
	optionID string,
	ctx Context,
) error {
	if ctx.State == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if handled, err := continuePendingExtended(pending, selected, optionID, ctx); handled {
		return err
	}
	switch pending.Kind {
	case PendingDestroyEnemyServant:
		if len(selected) != 1 {
			return fmt.Errorf("destroy pending choice requires exactly one target")
		}
		if ctx.DestroyField == nil {
			return fmt.Errorf("DestroyField callback is required")
		}
		return ctx.DestroyField(selected[0])
	case PendingBanishAlliedServant:
		if len(selected) != 1 {
			return fmt.Errorf("banish pending choice requires exactly one target")
		}
		if err := ops.Banish(ctx.State, selected[0]); err != nil {
			return err
		}
		switch pending.Payload {
		case "then_draw":
			return ops.Draw(ctx.State, ctx.Controller, 1)
		case "cosmetic_doctor":
			return resolveCosmeticDoctorDig(ctx)
		case "azathoth":
			enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
			if err != nil {
				return err
			}
			eidx, err := playerIdx(ctx.State, enemy)
			if err != nil {
				return err
			}
			if len(ctx.State.Players[eidx].Orbs) == 0 {
				ctx.State.MatchStatus = model.StatusFinished
				ctx.State.Result = model.MatchResult{
					Winner: ctx.Controller,
					Loser:  enemy,
					Reason: model.EndReasonZeroOrbs,
				}
			}
			return nil
		default:
			return nil
		}
	case PendingDiscardOneFromHand:
		if len(selected) != 1 {
			return fmt.Errorf("discard pending choice requires exactly one card")
		}
		return ops.Discard(ctx.State, selected[0])
	case PendingDiscardTwoFromHand:
		if len(selected) != 2 {
			return fmt.Errorf("discard pending choice requires exactly two cards")
		}
		for _, cardID := range selected {
			if err := ops.Discard(ctx.State, cardID); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func requireState(ctx Context) error {
	if ctx.State == nil {
		return fmt.Errorf("state cannot be nil")
	}
	return nil
}

func enemyPlayerID(state *model.MatchState, controller model.PlayerID) (model.PlayerID, error) {
	for _, player := range state.Players {
		if player.ID != controller {
			return player.ID, nil
		}
	}
	return "", fmt.Errorf("enemy player not found")
}

func playerIdx(state *model.MatchState, id model.PlayerID) (int, error) {
	for i, player := range state.Players {
		if player.ID == id {
			return i, nil
		}
	}
	return -1, fmt.Errorf("player %q not found", id)
}

func printedCost(ctx Context, instance model.CardInstance) (int, bool) {
	if ctx.Catalog == nil {
		return 0, false
	}
	def, ok := ctx.Catalog.FindByID(string(instance.CardID))
	if !ok {
		return 0, false
	}
	cost, err := strconv.Atoi(strings.TrimSpace(def.CostLevel))
	if err != nil {
		return 0, false
	}
	return cost, true
}

func isServantInstance(ctx Context, instance model.CardInstance) bool {
	switch printedType(ctx, instance) {
	case "":
		// No catalog: treat Servant-zone cards as servants.
		return true
	case "servant":
		return true
	default:
		// Barriers (and other types) share ServantZone storage but are not servants.
		return false
	}
}

func isBarrierInstance(ctx Context, instance model.CardInstance) bool {
	return printedType(ctx, instance) == "barrier"
}

// zoneServants returns Servant-type cards in playerIndex's servant zone.
func zoneServants(ctx Context, playerIndex int) []model.MatchCardID {
	if ctx.State == nil || playerIndex < 0 || playerIndex > 1 {
		return nil
	}
	var out []model.MatchCardID
	for _, id := range ctx.State.Players[playerIndex].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || !isServantInstance(ctx, inst) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func filterServants(ctx Context, ids []model.MatchCardID) []model.MatchCardID {
	out := make([]model.MatchCardID, 0, len(ids))
	for _, id := range ids {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || !isServantInstance(ctx, inst) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func countZoneServants(ctx Context, playerIndex int) int {
	return len(zoneServants(ctx, playerIndex))
}

func printedType(ctx Context, instance model.CardInstance) string {
	if ctx.Catalog == nil {
		return ""
	}
	def, ok := ctx.Catalog.FindByID(string(instance.CardID))
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(def.Type))
}

func offerOrSkip(ctx Context, choice model.PendingChoice) error {
	// Skip only when there is nothing to decide (no targets and no mode options).
	if len(choice.SelectableCardIDs) == 0 && len(choice.Options) == 0 {
		return nil
	}
	if ctx.OfferChoice == nil {
		return fmt.Errorf("OfferChoice callback is required for targeting effects")
	}
	return ctx.OfferChoice(choice)
}

func collectEnemyServants(ctx Context, maxCost int, restedOnly bool) ([]model.MatchCardID, error) {
	enemyID, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return nil, err
	}
	enemyIndex, err := playerIdx(ctx.State, enemyID)
	if err != nil {
		return nil, err
	}
	var out []model.MatchCardID
	for _, id := range ctx.State.Players[enemyIndex].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || !isServantInstance(ctx, inst) {
			continue
		}
		if restedOnly && inst.Orientation != model.OrientationRested {
			continue
		}
		if maxCost >= 0 {
			cost, ok := printedCost(ctx, inst)
			if !ok || cost > maxCost {
				continue
			}
		}
		out = append(out, id)
	}
	return out, nil
}

func collectAlliedServants(ctx Context) []model.MatchCardID {
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return nil
	}
	var out []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		if id == ctx.Source {
			continue
		}
		inst, ok := ctx.State.CardInstances[id]
		if !ok || !isServantInstance(ctx, inst) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func destroyAllServantsCostAtMost(ctx Context, maxCost int) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.DestroyField == nil {
		return fmt.Errorf("DestroyField callback is required")
	}
	var targets []model.MatchCardID
	for _, player := range ctx.State.Players {
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || !isServantInstance(ctx, inst) {
				continue
			}
			cost, ok := printedCost(ctx, inst)
			if !ok || cost > maxCost {
				continue
			}
			targets = append(targets, id)
		}
	}
	for _, id := range targets {
		if err := ctx.DestroyField(id); err != nil {
			return err
		}
	}
	return nil
}

func offerDestroyEnemy(ctx Context, maxCost int, restedOnly bool, prompt string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, maxCost, restedOnly)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDestroyEnemyServant,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

// --- Enter ---

func resolveCashCatEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	for _, player := range ctx.State.Players {
		if err := ops.Draw(ctx.State, player.ID, 1); err != nil {
			return fmt.Errorf("Cash Cat enter draw for %q: %w", player.ID, err)
		}
	}
	return nil
}

func resolveBloomLadyEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	n := len(hand)
	for _, id := range hand {
		if err := ops.Discard(ctx.State, id); err != nil {
			return err
		}
	}
	if n == 0 {
		return nil
	}
	return ops.Draw(ctx.State, ctx.Controller, n)
}

func resolveDestroyEnemyServantCostAtMost3(ctx Context) error {
	return offerDestroyEnemy(ctx, 3, false, "Destroy target enemy servant with cost 3 or less")
}

func resolveDestroyRestedEnemyServant(ctx Context) error {
	return offerDestroyEnemy(ctx, -1, true, "Destroy target rested enemy servant")
}

func resolveDestroyTargetBarrier(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, player := range ctx.State.Players {
		if player.ID == ctx.Controller {
			continue
		}
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || !isBarrierInstance(ctx, inst) {
				continue
			}
			targets = append(targets, id)
		}
	}
	// Also allow allied barriers when text says "target barrier" without enemy.
	for _, player := range ctx.State.Players {
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || !isBarrierInstance(ctx, inst) {
				continue
			}
			if !containsID(targets, id) {
				targets = append(targets, id)
			}
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDestroyEnemyServant,
		Prompt:            "Destroy target barrier",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func containsID(ids []model.MatchCardID, want model.MatchCardID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func resolveBanishAlliedServantTarget(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishAlliedServant,
		Prompt:            "Banish an allied servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: collectAlliedServants(ctx),
	})
}

func resolveOuroborosEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 1); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	if len(hand) == 0 {
		return nil
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDiscardOneFromHand,
		Prompt:            "Discard a card",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: hand,
	})
}

func resolveNONEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 2); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	if len(hand) < 2 {
		// Discard as many as possible (golden rule).
		for _, id := range hand {
			if err := ops.Discard(ctx.State, id); err != nil {
				return err
			}
		}
		return nil
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDiscardTwoFromHand,
		Prompt:            "Discard two cards",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: hand,
	})
}

// --- Last Words / Rest / EOT ---

func resolveInfiniCoinLastWords(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.MoveToHand(ctx.State, ctx.Source)
}

func resolveTwinkleCharmRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.Draw(ctx.State, ctx.Controller, 1)
}

func resolveAiriMill3Rest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.Mill(ctx.State, ctx.Controller, 3)
}

func resolveMeltyIceCreamEOT(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.Banish(ctx.State, ctx.Source)
}

func resolveShoppingPlazaEOT(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	// At end of enemy turn: Draw a card.
	if ctx.TurnPlayer == "" || ctx.TurnPlayer == ctx.Controller {
		return nil
	}
	return ops.Draw(ctx.State, ctx.Controller, 1)
}

func resolveBottomSelfEnemyTurn(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.TurnPlayer == "" || ctx.TurnPlayer == ctx.Controller {
		return nil
	}
	return ops.ReturnToDeck(ctx.State, ctx.Source, model.DeckPlacementBottom)
}

// --- Conjure HookResolve ---

func resolveDraw1(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 1); err != nil {
		return err
	}
	noteConjureDraws(ctx, 1)
	return nil
}

func resolveDraw2(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 2); err != nil {
		return err
	}
	noteConjureDraws(ctx, 2)
	return nil
}

func resolveDraw3(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 3); err != nil {
		return err
	}
	noteConjureDraws(ctx, 3)
	return nil
}

func noteConjureDraws(ctx Context, count int) {
	if ctx.State == nil || ctx.Hook != HookResolve || count <= 0 {
		return
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return
	}
	ctx.State.TurnFlags.ConjureEffectDraws[idx] += count
}

func resolveDestroyAllServantsCostAtMost2(ctx Context) error {
	return destroyAllServantsCostAtMost(ctx, 2)
}

func resolveDestroyAllServantsCostAtMost5(ctx Context) error {
	return destroyAllServantsCostAtMost(ctx, 5)
}

func resolveReturnAllServantsToHand(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, player := range ctx.State.Players {
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || !isServantInstance(ctx, inst) {
				continue
			}
			targets = append(targets, id)
		}
	}
	for _, id := range targets {
		if err := ops.ReturnToHand(ctx.State, id); err != nil {
			return err
		}
	}
	return nil
}

func resolveRestAllEnemyServants(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemyID, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, enemyID)
	if err != nil {
		return err
	}
	for _, id := range append([]model.MatchCardID(nil), ctx.State.Players[idx].ServantZone...) {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || !isServantInstance(ctx, inst) {
			continue
		}
		if err := ops.SetOrientation(ctx.State, id, model.OrientationRested); err != nil {
			return err
		}
	}
	return nil
}

func resolveTopDeckToOrb(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	if len(ctx.State.Players[idx].Deck) == 0 {
		return nil
	}
	top := ctx.State.Players[idx].Deck[0]
	return ops.PutAsOrb(ctx.State, top, ctx.Controller)
}

func resolvePlayNice(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	ctx.State.ContinuousUntilEOT.PlayerCannotBeAttacked[idx] = true
	return ops.Exile(ctx.State, ctx.Source)
}
