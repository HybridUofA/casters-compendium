package effects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Additional pending kinds for bulk automation.
const (
	PendingReturnServantHand   = "fx:return_servant_hand"
	PendingReturnTwoServants   = "fx:return_two_servants"
	PendingGainControlServant  = "fx:gain_control_servant"
	PendingOrientRest          = "fx:orient_rest"
	PendingOrientReverse       = "fx:orient_reverse"
	PendingOrientRestOrReverse = "fx:orient_rest_or_reverse"
	PendingSearchDeckToHand    = "fx:search_deck_to_hand"
	PendingSearchDeckToField   = "fx:search_deck_to_field"
	PendingGYToHand            = "fx:gy_to_hand"
	PendingGYToField           = "fx:gy_to_field"
	PendingGYToDeckTop         = "fx:gy_to_deck_top"
	PendingHandToOrb           = "fx:hand_to_orb"
	PendingHandToOrbUpTo2      = "fx:hand_to_orb_upto2"
	PendingModifyATK           = "fx:modify_atk"
	PendingModifyDEF           = "fx:modify_def"
	PendingGrantDC             = "fx:grant_dc"
	PendingExileTarget         = "fx:exile_target"
	PendingPutAsCasterFD       = "fx:put_as_caster_fd"
	PendingPeekOrb             = "fx:peek_orb"
	PendingBanishEnemyServant  = "fx:banish_enemy_servant"
	PendingFaceDownCaster      = "fx:face_down_caster"
	PendingCorruptOrb          = "fx:corrupt_orb"
	PendingBanishOrb           = "fx:banish_orb"
)

func continuePendingExtended(
	pending model.PendingChoice,
	selected []model.MatchCardID,
	optionID string,
	ctx Context,
) (handled bool, err error) {
	if handled, err := continueUniquePending(pending, selected, ctx); handled {
		return true, err
	}
	switch pending.Kind {
	case PendingHildeDeckBottom:
		return true, continueHildeDeckBottom(pending, optionID, ctx)
	case PendingRPSChoice:
		return true, continueRPSChoice(pending, optionID, ctx)
	case PendingReturnServantHand:
		if len(selected) != 1 {
			return true, fmt.Errorf("return pending requires one target")
		}
		noteEnemyReturned(ctx.State, selected[0])
		if err := ops.ReturnToHand(ctx.State, selected[0]); err != nil {
			return true, err
		}
		if pending.Payload == "then_allied_atk_500" {
			return true, resolveAlliedATK500(ctx)
		}
		return true, nil
	case PendingReturnTwoServants:
		if len(selected) != 1 {
			return true, fmt.Errorf("return-two pending selects one servant at a time")
		}
		noteEnemyReturned(ctx.State, selected[0])
		if err := ops.ReturnToHand(ctx.State, selected[0]); err != nil {
			return true, err
		}
		// Chain second pick when payload says so.
		if pending.Payload == "need_second" {
			remaining := filterOut(pending.SelectableCardIDs, selected[0])
			if len(remaining) == 0 {
				return true, nil
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          pending.PlayerID,
				Kind:              PendingReturnServantHand,
				Prompt:            "Return another servant to its owner's hand",
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: remaining,
			})
		}
		return true, nil
	case PendingGainControlServant:
		if len(selected) != 1 {
			return true, fmt.Errorf("gain control requires one target")
		}
		return true, ops.GainControl(ctx.State, selected[0], ctx.Controller)
	case PendingOrientRest:
		if len(selected) != 1 {
			return true, fmt.Errorf("rest pending requires one target")
		}
		return true, ops.SetOrientation(ctx.State, selected[0], model.OrientationRested)
	case PendingOrientReverse:
		if len(selected) != 1 {
			return true, fmt.Errorf("reverse pending requires one target")
		}
		return true, ops.SetOrientation(ctx.State, selected[0], model.OrientationReversed)
	case PendingOrientRestOrReverse:
		if len(selected) != 1 {
			return true, fmt.Errorf("orient pending requires one target")
		}
		switch optionID {
		case "rest":
			return true, ops.SetOrientation(ctx.State, selected[0], model.OrientationRested)
		case "reverse":
			return true, ops.SetOrientation(ctx.State, selected[0], model.OrientationReversed)
		default:
			return true, fmt.Errorf("choose Rest or Reverse")
		}
	case PendingSearchDeckToHand:
		if len(selected) != 1 {
			return true, fmt.Errorf("search pending requires one card")
		}
		if err := ops.MoveFromDeckToHand(ctx.State, ctx.Controller, selected[0]); err != nil {
			return true, err
		}
		return true, shuffleControllerDeck(ctx)
	case PendingSearchDeckToField:
		if len(selected) != 1 {
			return true, fmt.Errorf("search-to-field requires one card")
		}
		orient := model.OrientationRecovered
		if strings.Contains(pending.Payload, "rest") {
			orient = model.OrientationRested
		}
		if err := ops.PutAsServant(ctx.State, selected[0], ctx.Controller, model.CardFaceUp, orient); err != nil {
			return true, err
		}
		return true, shuffleControllerDeck(ctx)
	case PendingGYToHand:
		if len(selected) != 1 {
			return true, fmt.Errorf("graveyard-to-hand requires one card")
		}
		return true, ops.MoveToHand(ctx.State, selected[0])
	case PendingGYToField:
		if len(selected) != 1 {
			return true, fmt.Errorf("graveyard-to-field requires one card")
		}
		return true, ops.PutAsServant(ctx.State, selected[0], ctx.Controller, model.CardFaceUp, model.OrientationRecovered)
	case PendingGYToDeckTop:
		if len(selected) != 1 {
			return true, fmt.Errorf("graveyard-to-deck requires one card")
		}
		return true, ops.ReturnToDeck(ctx.State, selected[0], model.DeckPlacementTop)
	case PendingHandToOrb, PendingHandToOrbUpTo2:
		if len(selected) != 1 {
			return true, fmt.Errorf("hand-to-orb requires one card")
		}
		if err := ops.PutAsOrb(ctx.State, selected[0], ctx.Controller); err != nil {
			return true, err
		}
		if pending.Kind == PendingHandToOrbUpTo2 && pending.Payload != "done_second" {
			idx, err := playerIdx(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
			if len(hand) == 0 {
				return true, nil
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          ctx.Controller,
				Kind:              PendingHandToOrb,
				Prompt:            "You may put another hand card into an orb zone (or decline)",
				AllowDecline:      true,
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: hand,
				Payload:           "done_second",
			})
		}
		return true, nil
	case PendingModifyATK:
		if len(selected) != 1 {
			return true, fmt.Errorf("modify ATK requires one target")
		}
		payload := pending.Payload
		grantDC := false
		if strings.HasSuffix(payload, "+dc") {
			grantDC = true
			payload = strings.TrimSuffix(payload, "+dc")
		}
		delta, _ := strconv.Atoi(payload)
		if err := ops.ModifyAttack(ctx.State, selected[0], delta); err != nil {
			return true, err
		}
		if grantDC {
			return true, ops.GrantDoubleCorrupt(ctx.State, selected[0], true)
		}
		return true, nil
	case PendingModifyDEF:
		if len(selected) != 1 {
			return true, fmt.Errorf("modify DEF requires one target")
		}
		delta, _ := strconv.Atoi(pending.Payload)
		return true, ops.ModifyDefense(ctx.State, selected[0], delta)
	case PendingGrantDC:
		if len(selected) != 1 {
			return true, fmt.Errorf("grant DC requires one target")
		}
		return true, ops.GrantDoubleCorrupt(ctx.State, selected[0], true)
	case PendingExileTarget:
		if len(selected) != 1 {
			return true, fmt.Errorf("exile requires one target")
		}
		if err := ops.Exile(ctx.State, selected[0]); err != nil {
			return true, err
		}
		switch pending.Payload {
		case "return_eot":
			ctx.State.ExileReturnAtEOT = append(ctx.State.ExileReturnAtEOT, selected[0])
			return true, nil
		case "then_enemy_draw":
			enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			return true, ops.Draw(ctx.State, enemy, 1)
		case "then_exile_own_fu":
			idx, err := playerIdx(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			var targets []model.MatchCardID
			for _, id := range append(append([]model.MatchCardID{}, ctx.State.Players[idx].ServantZone...), ctx.State.Players[idx].CasterZone...) {
				inst, ok := ctx.State.CardInstances[id]
				if ok && inst.Face == model.CardFaceUp {
					targets = append(targets, id)
				}
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          ctx.Controller,
				Kind:              PendingExileTarget,
				Prompt:            "Remove a face-up card you control from the game",
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: targets,
			})
		default:
			return true, nil
		}
	case PendingPutAsCasterFD:
		if len(selected) != 1 {
			return true, fmt.Errorf("put-as-caster requires one target")
		}
		if pending.Payload == "furious" {
			if err := ops.PutAsServant(ctx.State, selected[0], ctx.Controller, model.CardFaceUp, model.OrientationRecovered); err != nil {
				return true, err
			}
			_ = ops.GrantDoubleCorrupt(ctx.State, selected[0], true)
			_ = ops.SetCombatModifiers(ctx.State, selected[0], 5000, 5000) // absolute-ish via modifiers if printed low
			return true, nil
		}
		owner := ctx.State.CardInstances[selected[0]].Owner
		return true, ops.PutAsCaster(ctx.State, selected[0], owner, model.CardFaceDown, model.OrientationRecovered)
	case PendingPeekOrb:
		if len(selected) != 1 {
			return true, fmt.Errorf("peek orb requires one orb")
		}
		model.MarkCardKnown(ctx.State, ctx.Controller, selected[0])
		if pending.Payload == "then_draw" {
			return true, ops.Draw(ctx.State, ctx.Controller, 1)
		}
		return true, nil
	case PendingCorruptOrb:
		if len(selected) != 1 {
			return true, fmt.Errorf("corrupt orb requires one orb")
		}
		return true, corruptOrbByID(ctx, selected[0])
	case PendingBanishOrb:
		if len(selected) != 1 {
			return true, fmt.Errorf("banish orb requires one orb")
		}
		if err := ops.Banish(ctx.State, selected[0]); err != nil {
			return true, err
		}
		if pending.Payload == "then_corrupt_orb" {
			return true, offerCorruptTargetOrb(ctx)
		}
		return true, nil
	case PendingBanishEnemyServant:
		if len(selected) != 1 {
			return true, fmt.Errorf("banish enemy requires one target")
		}
		return true, ops.Banish(ctx.State, selected[0])
	case PendingFaceDownCaster:
		if len(selected) != 1 {
			return true, fmt.Errorf("face-down caster requires one target")
		}
		return true, ops.SetFace(ctx.State, selected[0], model.CardFaceDown)
	case PendingBanishStockThen:
		if len(selected) != 1 {
			return true, fmt.Errorf("banish stock requires one card")
		}
		if err := ops.BanishStock(ctx.State, selected[0]); err != nil {
			return true, err
		}
		switch pending.Payload {
		case "destroy_enemy_3":
			return true, resolveDestroyEnemyServantCostAtMost3(ctx)
		case "atk_1000":
			return true, resolveAlliedATK1000(ctx)
		case "draw_1":
			return true, ops.Draw(ctx.State, ctx.Controller, 1)
		case "arthur_top2_casters":
			return true, applyArthurTop2Casters(ctx)
		case "maria_search_band":
			return true, applyMariaSearchBand(ctx)
		case "ran_exile_eot":
			return true, applyRanExileEOT(ctx)
		default:
			return true, fmt.Errorf("unknown banish-stock follow-up %q", pending.Payload)
		}
	case PendingDiscardCasterThen:
		if len(selected) != 1 {
			return true, fmt.Errorf("discard caster requires one card")
		}
		if err := ops.Discard(ctx.State, selected[0]); err != nil {
			return true, err
		}
		switch pending.Payload {
		case "destroy_enemy_3":
			return true, resolveDestroyEnemyServantCostAtMost3(ctx)
		case "reverse_enemy":
			return true, resolveReverseEnemyServant(ctx)
		case "grant_dc":
			return true, resolveGrantAlliedDC(ctx)
		case "atk_1000":
			return true, resolveAlliedATK1000(ctx)
		case "draw_1":
			return true, ops.Draw(ctx.State, ctx.Controller, 1)
		case "gy_supplies_hand":
			return true, resolveSearchSuppliesFromGY(ctx)
		case "gy_insect_hand":
			return true, resolveSearchInsectFromGY(ctx)
		case "grant_slow_start":
			return true, resolveGrantEnemySlowStart(ctx)
		case "destroy_reversed_if_corrupt":
			idx, err := playerIdx(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			if ctx.State.TurnFlags.AlliedCorruptedOrb[idx] < 1 {
				return true, nil
			}
			return true, offerDestroyEnemy(ctx, -1, false, "Destroy target reversed enemy servant")
		default:
			return true, fmt.Errorf("unknown discard-caster follow-up %q", pending.Payload)
		}
	case PendingDiscardOneFromHand:
		if strings.HasPrefix(pending.Payload, "discard_left:") || pending.Payload == "chain_one_more" {
			if len(selected) != 1 {
				return true, fmt.Errorf("discard requires one card")
			}
			discarder := ctx.Controller
			if inst, ok := ctx.State.CardInstances[selected[0]]; ok {
				discarder = inst.Controller
			}
			if err := ops.Discard(ctx.State, selected[0]); err != nil {
				return true, err
			}
			noteEnemyDiscarded(ctx.State, discarder)
			left := 0
			if strings.HasPrefix(pending.Payload, "discard_left:") {
				left, _ = strconv.Atoi(strings.TrimPrefix(pending.Payload, "discard_left:"))
			} else {
				left = 1
			}
			if left <= 0 {
				return true, nil
			}
			return true, offerDiscardChain(ctx, left)
		}
		return false, nil
	case PendingStaplerPick:
		if len(selected) != 1 {
			return true, fmt.Errorf("stapler pick requires one target")
		}
		switch {
		case pending.Payload == "need_second":
			targets, err := collectEnemyServants(ctx, -1, false)
			if err != nil {
				return true, err
			}
			remaining := filterOut(targets, selected[0])
			if len(remaining) == 0 {
				// Only one legal target: enemy banishes that one.
				enemyID, err := enemyPlayerID(ctx.State, ctx.Controller)
				if err != nil {
					return true, err
				}
				return true, offerOrSkip(ctx, model.PendingChoice{
					PlayerID:          enemyID,
					Kind:              PendingBanishEnemyServant,
					Prompt:            "Banish the chosen servant",
					SourceCardID:      pending.SourceCardID,
					SelectableCardIDs: []model.MatchCardID{selected[0]},
				})
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          pending.PlayerID,
				Kind:              PendingStaplerPick,
				Prompt:            "Choose a second enemy servant",
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: remaining,
				Payload:           "first:" + string(selected[0]),
			})
		case strings.HasPrefix(pending.Payload, "first:"):
			first := model.MatchCardID(strings.TrimPrefix(pending.Payload, "first:"))
			second := selected[0]
			enemyID, err := enemyPlayerID(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          enemyID,
				Kind:              PendingBanishEnemyServant,
				Prompt:            "Banish one of the chosen servants",
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: []model.MatchCardID{first, second},
			})
		default:
			return true, fmt.Errorf("unknown stapler payload %q", pending.Payload)
		}
	case PendingRestEnemyServant:
		if len(selected) != 1 {
			return true, fmt.Errorf("rest enemy requires one target")
		}
		if err := ops.SetOrientation(ctx.State, selected[0], model.OrientationRested); err != nil {
			return true, err
		}
		if pending.Payload == "need_second_rest" {
			targets, err := collectEnemyServants(ctx, -1, false)
			if err != nil {
				return true, err
			}
			remaining := filterOut(targets, selected[0])
			if len(remaining) == 0 {
				return true, nil
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          pending.PlayerID,
				Kind:              PendingRestEnemyServant,
				Prompt:            "Rest another enemy servant",
				AllowDecline:      true,
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: remaining,
			})
		}
		return true, nil
	case PendingRestEnemyCasterLv1, PendingRecoverAlliedCaster:
		if len(selected) != 1 {
			return true, fmt.Errorf("caster orient requires one target")
		}
		orient := model.OrientationRested
		if pending.Kind == PendingRecoverAlliedCaster {
			orient = model.OrientationRecovered
		}
		return true, ops.SetOrientation(ctx.State, selected[0], orient)
	case PendingSwapATKDEF:
		if len(selected) != 1 {
			return true, fmt.Errorf("swap requires one target")
		}
		return true, applyPrintedStatSwap(ctx, selected[0])
	case PendingGrantSlowStart:
		if len(selected) != 1 {
			return true, fmt.Errorf("slow start requires one target")
		}
		inst := ctx.State.CardInstances[selected[0]]
		inst.GrantedSlowStartUntilEOT = true
		ctx.State.CardInstances[selected[0]] = inst
		return true, nil
	case PendingLoseSlowStart:
		if len(selected) != 1 {
			return true, fmt.Errorf("lose slow start requires one target")
		}
		inst := ctx.State.CardInstances[selected[0]]
		inst.LosesSlowStartUntilEOT = true
		ctx.State.CardInstances[selected[0]] = inst
		return true, nil
	case PendingServantToOrb:
		if len(selected) != 1 {
			return true, fmt.Errorf("servant-to-orb requires one target")
		}
		inst := ctx.State.CardInstances[selected[0]]
		return true, ops.PutAsOrb(ctx.State, selected[0], inst.Owner)
	case PendingExchangeServants:
		if len(selected) != 1 {
			return true, fmt.Errorf("exchange requires one target")
		}
		if pending.Payload == "need_enemy" {
			targets, err := collectEnemyServants(ctx, -1, false)
			if err != nil {
				return true, err
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          pending.PlayerID,
				Kind:              PendingExchangeServants,
				Prompt:            "Choose an enemy servant to exchange",
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: targets,
				Payload:           "ally:" + string(selected[0]),
			})
		}
		if strings.HasPrefix(pending.Payload, "ally:") {
			ally := model.MatchCardID(strings.TrimPrefix(pending.Payload, "ally:"))
			return true, ops.ExchangeControl(ctx.State, ally, selected[0])
		}
		return true, nil
	case PendingEnemyGYToBottom:
		if len(selected) != 1 {
			return true, fmt.Errorf("enemy gy to bottom requires one card")
		}
		return true, ops.ReturnToDeck(ctx.State, selected[0], model.DeckPlacementBottom)
	case PendingDestroyAnyServant:
		if len(selected) != 1 {
			return true, fmt.Errorf("destroy requires one target")
		}
		if ctx.DestroyField == nil {
			return true, fmt.Errorf("DestroyField required")
		}
		if strings.HasPrefix(pending.Payload, "little_red:") {
			// Accumulate then on decline — for Confirm path destroy immediately if single pick playtest.
			return true, ctx.DestroyField(selected[0])
		}
		return true, ctx.DestroyField(selected[0])
	case PendingUnityRestAllies:
		totalATK := 0
		for _, id := range selected {
			inst, ok := ctx.State.CardInstances[id]
			if !ok {
				continue
			}
			_ = ops.SetOrientation(ctx.State, id, model.OrientationRested)
			if ctx.Catalog != nil {
				if def, ok := ctx.Catalog.FindByID(string(inst.CardID)); ok {
					printed, _ := strconv.Atoi(strings.TrimSpace(def.Attack))
					totalATK += printed + inst.AttackModifier
				}
			}
		}
		if totalATK != 0 {
			_ = ops.ModifyAttack(ctx.State, pending.SourceCardID, totalATK)
		}
		return true, nil
	case PendingReorderDeckTops:
		if len(selected) != 1 {
			return true, fmt.Errorf("reorder requires choosing the new top card")
		}
		parts := strings.Split(pending.Payload, ",")
		ordered := []model.MatchCardID{selected[0]}
		for _, p := range parts {
			id := model.MatchCardID(p)
			if id == "" || id == selected[0] {
				continue
			}
			ordered = append(ordered, id)
		}
		return true, ops.ReorderDeckTops(ctx.State, ctx.Controller, ordered)
	case PendingReturnLv1CasterThen:
		if len(selected) != 1 {
			return true, fmt.Errorf("return caster requires one target")
		}
		if err := ops.ReturnToHand(ctx.State, selected[0]); err != nil {
			return true, err
		}
		if pending.Payload == "atk_1000" {
			return true, resolveAlliedATK1000(ctx)
		}
		return true, nil
	case PendingBanishFDCasterThen:
		if len(selected) != 1 {
			return true, fmt.Errorf("banish fd caster requires one target")
		}
		if err := ops.Banish(ctx.State, selected[0]); err != nil {
			return true, err
		}
		switch pending.Payload {
		case "destroy_enemy_3":
			return true, resolveDestroyEnemyServantCostAtMost3(ctx)
		case "destroy_enemy_2":
			return true, resolveDestroyEnemyCostAtMost2(ctx)
		default:
			return true, nil
		}
	case PendingHandServantToField:
		if len(selected) != 1 {
			return true, fmt.Errorf("hand to field requires one card")
		}
		if err := ops.PutAsServant(ctx.State, selected[0], ctx.Controller, model.CardFaceUp, model.OrientationRecovered); err != nil {
			return true, err
		}
		if pending.Payload == "need_second" {
			idx, err := playerIdx(ctx.State, ctx.Controller)
			if err != nil {
				return true, err
			}
			var matches []model.MatchCardID
			for _, id := range ctx.State.Players[idx].Hand {
				inst, ok := ctx.State.CardInstances[id]
				if !ok || printedType(ctx, inst) != "servant" {
					continue
				}
				cost, ok := printedCost(ctx, inst)
				if ok && cost <= 2 {
					matches = append(matches, id)
				}
			}
			return true, offerOrSkip(ctx, model.PendingChoice{
				PlayerID:          pending.PlayerID,
				Kind:              PendingHandServantToField,
				Prompt:            "You may put another cost ≤2 servant from hand into the field",
				AllowDecline:      true,
				SourceCardID:      pending.SourceCardID,
				SelectableCardIDs: matches,
			})
		}
		return true, nil
	default:
		return false, nil
	}
}

func noteEnemyDiscarded(state *model.MatchState, discardingPlayer model.PlayerID) {
	if state == nil {
		return
	}
	for index, player := range state.Players {
		if player.ID == discardingPlayer {
			state.TurnFlags.EnemyDiscarded[1-index]++
			return
		}
	}
}

func noteEnemyReturned(state *model.MatchState, cardID model.MatchCardID) {
	if state == nil {
		return
	}
	instance, ok := state.CardInstances[cardID]
	if !ok {
		return
	}
	for index, player := range state.Players {
		if player.ID == instance.Controller {
			state.TurnFlags.EnemyReturnedToHand[1-index]++
			return
		}
	}
}

func filterOut(ids []model.MatchCardID, remove model.MatchCardID) []model.MatchCardID {
	out := make([]model.MatchCardID, 0, len(ids))
	for _, id := range ids {
		if id != remove {
			out = append(out, id)
		}
	}
	return out
}

func shuffleControllerDeck(ctx Context) error {
	if ctx.State == nil || ctx.State.EffectRandom == nil {
		return nil
	}
	return ops.ShuffleDeck(ctx.State, ctx.Controller, ctx.State.EffectRandom)
}
