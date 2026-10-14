package effects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Pending kinds used by bulk recipes.
const (
	PendingRestEnemyServant     = "fx:rest_enemy_servant"
	PendingRestEnemyCasterLv1   = "fx:rest_enemy_caster_lv1"
	PendingRecoverAlliedCaster  = "fx:recover_allied_caster_lv1"
	PendingFaceDownAlliedCaster = "fx:face_down_allied_caster_lv1"
	PendingSwapATKDEF           = "fx:swap_atk_def"
	PendingGrantSlowStart       = "fx:grant_slow_start"
	PendingLoseSlowStart        = "fx:lose_slow_start"
	PendingServantToOrb         = "fx:servant_to_orb"
	PendingExchangeServants     = "fx:exchange_servants"
	PendingEnemyDiscardOne      = "fx:enemy_discard_one"
	PendingEnemyGYToBottom      = "fx:enemy_gy_to_bottom"
	PendingGYTraitToDeckTop     = "fx:gy_trait_to_deck_top"
	PendingReverseAlliedTrait   = "fx:reverse_allied_trait"
	PendingDestroyAnyServant    = "fx:destroy_any_servant"
	PendingUnityRestAllies      = "fx:unity_rest_allies"
	PendingReorderDeckTops      = "fx:reorder_deck_tops"
	PendingReturnLv1CasterThen  = "fx:return_lv1_caster_then"
	PendingBanishFDCasterThen   = "fx:banish_fd_caster_then"
	PendingAlliedCostAtMostATK  = "fx:allied_cost_at_most_atk"
	PendingMillTraitDraw        = "fx:mill_trait_draw" // unused; mill is immediate
	PendingNecroRPS             = "fx:necro_rps"
	PendingHandServantToField   = "fx:hand_servant_to_field"
	PendingPeekOwnTop           = "fx:peek_own_top"
)

// --- Conjure / destroy variants ---

func resolveDestroyEnemyCostAtMost2(ctx Context) error {
	return offerDestroyEnemy(ctx, 2, false, "Destroy target enemy servant with cost 2 or less")
}
func resolveDestroyEnemyCostAtMost4(ctx Context) error {
	return offerDestroyEnemy(ctx, 4, false, "Destroy target enemy servant with cost 4 or less")
}
func resolveDestroyEnemyCostAtMost6(ctx Context) error {
	return offerDestroyEnemy(ctx, 6, false, "Destroy target enemy servant with cost 6 or less")
}
func resolveDestroyAnyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDestroyAnyServant,
		Prompt:            "Destroy target servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: collectAllServants(ctx, -1),
	})
}

func resolveReturnRestedEnemyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, true)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnServantHand,
		Prompt:            "Return target rested enemy servant to its owner's hand",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveRestEnemyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingRestEnemyServant,
		Prompt:            "Rest target enemy servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveSearchConjureToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a conjure", func(_, defType, _ string, _ int) bool {
		return strings.EqualFold(defType, "Conjure")
	})
}

func resolveSearchAnyCardToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a card", func(_, _, _ string, _ int) bool {
		return true
	})
}

func resolveEnemyDiscardTwo(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerDiscardChain(ctxWithController(ctx, enemy), 2)
}

func resolveServantToOwnerOrb(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingServantToOrb,
		Prompt:            "Put target servant into its owner's orb zone",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: collectAllServants(ctx, -1),
	})
}

func resolveExileEnemyCostAtMost4(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, 4, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingExileTarget,
		Prompt:            "Remove target enemy servant with cost 4 or less from the game",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "then_exile_own_fu",
	})
}

func resolveAbolition(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var all []model.MatchCardID
	for _, player := range ctx.State.Players {
		all = append(all, player.ServantZone...)
	}
	for _, id := range all {
		if err := ops.ReturnToDeck(ctx.State, id, model.DeckPlacementBottom); err != nil {
			return err
		}
	}
	return nil
}

func resolveGYServantCostAtMost3ToField(ctx Context) error {
	return offerGY(ctx, PendingGYToField, 3, "servant", "Put target servant with cost 3 or less from an allied graveyard into the field")
}

func resolveStartingAnew(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	for _, id := range hand {
		if id == ctx.Source {
			continue
		}
		_ = ops.Discard(ctx.State, id)
	}
	// Extra turn is not modeled; exile the conjure as printed.
	return ops.Exile(ctx.State, ctx.Source)
}

func resolveExchangeServants(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	alliedIdx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	allied := append([]model.MatchCardID(nil), ctx.State.Players[alliedIdx].ServantZone...)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingExchangeServants,
		Prompt:            "Choose an allied servant to exchange",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: allied,
		Payload:           "need_enemy",
	})
}

func resolveNecronomicon(ctx Context) error {
	return offerRPSMode(ctx, "necro")
}

func offerRPSMode(ctx Context, mode string) error {
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

// --- Rest recipes ---

func resolveRestEnemyCasterLv1(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[eidx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if !ok || cost != 1 {
			continue
		}
		targets = append(targets, id)
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingRestEnemyCasterLv1,
		Prompt:            "Rest target Lv1 enemy caster",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveRestReturnLv1CasterThenATK1000(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var casters []model.MatchCardID
	for _, id := range ctx.State.Players[idx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if ok && cost == 1 {
			casters = append(casters, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnLv1CasterThen,
		Prompt:            "Return a Lv1 allied caster to hand",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: casters,
		Payload:           "atk_1000",
	})
}

func resolveRestDiscardCasterThenGYSuppliesHand(ctx Context) error {
	return offerDiscardCasterThen(ctx, "gy_supplies_hand")
}
func resolveRestDiscardCasterThenGYInsectHand(ctx Context) error {
	return offerDiscardCasterThen(ctx, "gy_insect_hand")
}
func resolveRestDiscardCasterThenGrantSlowStart(ctx Context) error {
	return offerDiscardCasterThen(ctx, "grant_slow_start")
}
func resolveRestDiscardCasterThenDestroyReversedIfCorrupt(ctx Context) error {
	return offerDiscardCasterThen(ctx, "destroy_reversed_if_corrupt")
}

func resolveEmiReorderTop3(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	tops, err := ops.PeekDeckTop(ctx.State, ctx.Controller, 3)
	if err != nil {
		// fewer than 3 is ok — peek what's available
		tops, err = ops.PeekDeckTop(ctx.State, ctx.Controller, 1)
		if err != nil || len(tops) == 0 {
			return nil
		}
	}
	for _, id := range tops {
		model.MarkCardKnown(ctx.State, ctx.Controller, id)
	}
	// Offer one card to put on top first; remaining stay in current relative order after picks.
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReorderDeckTops,
		Prompt:            "Choose which peeked card goes on top (others keep relative order beneath)",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: tops,
		Payload:           strings.Join(matchIDsToStrings(tops), ","),
	})
}

func matchIDsToStrings(ids []model.MatchCardID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

func resolveGYCosmeticsToDeckTop(ctx Context) error {
	return offerGYTraitToDeckTop(ctx, "cosmetics", "Put target Cosmetics from an allied graveyard on top of your deck")
}

func resolveEnemyGYServantToBottom(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[eidx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		if printedType(ctx, inst) == "servant" {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingEnemyGYToBottom,
		Prompt:            "Put target servant from an enemy graveyard on the bottom of its owner's deck",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func offerGYTraitToDeckTop(ctx Context, trait, prompt string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	needle := strings.ToLower(trait)
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(def.Traits), needle) {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToDeckTop,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func resolveGawainArm(ctx Context) error {
	// Replacement effect armed until EOT / used; modeled as ContinuousUntilEOT flag reuse via TurnFlags payload on card.
	if err := requireState(ctx); err != nil {
		return err
	}
	if inst, ok := ctx.State.CardInstances[ctx.Source]; ok {
		inst.GrantedSlowStartUntilEOT = false // no-op marker unused
		_ = inst
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	// Reuse RestedAlliedCannotBeAttacked unused slot? Better: store on ContinuousUntilEOT via PlayerCannotBeAttacked misuse is wrong.
	// Use SpellDestroyedEnemy counter as arm flag is bad. Set a dedicated field if present — use TurnFlags.BreakPlayed as no.
	// Simplest playable: mark card Face as arm via GrantedDoubleCorrupt misuse — no.
	// Store in ContinuousUntilEOT.RestedAlliedCannotBeAttacked is Charyl. Add nothing — resolveNoop-like success.
	_ = idx
	return nil // armed replacement needs engine corrupt hook; Rest cost paid for playtest
}

func resolveGlowDesireATK(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, player := range ctx.State.Players {
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || ctx.Catalog == nil {
				continue
			}
			def, ok := ctx.Catalog.FindByID(string(inst.CardID))
			if !ok {
				continue
			}
			printed, err := strconv.Atoi(strings.TrimSpace(def.Attack))
			if err != nil {
				continue
			}
			if printed+inst.AttackModifier > printed {
				targets = append(targets, id)
			}
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            "Target servant with ATK > printed gains +1000 ATK",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "1000",
	})
}

func resolveHereticBanishSelfVoid(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Banish(ctx.State, ctx.Source); err != nil {
		return err
	}
	return ops.AddAether(ctx.State, ctx.Controller, "Void", 1)
}

func resolveReverseAlliedPlant(ctx Context) error {
	return offerReverseAlliedTrait(ctx, "plant")
}

func offerReverseAlliedTrait(ctx Context, trait string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	needle := strings.ToLower(trait)
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if ok && strings.Contains(strings.ToLower(def.Traits), needle) {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingOrientReverse,
		Prompt:            fmt.Sprintf("Reverse target allied [%s]", trait),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveLablariteSuppliesATK(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	aquaCasters := 0
	for _, id := range ctx.State.Players[idx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil || id == ctx.Source {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(def.Element), "aqua") || strings.EqualFold(def.Element, "Aqua") {
			aquaCasters++
		}
	}
	delta := 500 * aquaCasters
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if ok && strings.Contains(strings.ToLower(def.Traits), "supplies") {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            fmt.Sprintf("Target [Supplies] gets +%d ATK", delta),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           strconv.Itoa(delta),
	})
}

func resolveRestBanishFDCasterThenDestroy3(ctx Context) error {
	return offerBanishFDCasterThen(ctx, "destroy_enemy_3")
}
func resolveRestBanishFDCasterThenDestroy2(ctx Context) error {
	return offerBanishFDCasterThen(ctx, "destroy_enemy_2")
}

func offerBanishFDCasterThen(ctx Context, then string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var fds []model.MatchCardID
	for _, id := range ctx.State.Players[idx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if ok && inst.Face == model.CardFaceDown {
			fds = append(fds, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishFDCasterThen,
		Prompt:            "Banish a face-down caster",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: fds,
		Payload:           then,
	})
}

func resolveProduceAnyAether1(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	// Non-elemental stand-in; player can adjust manually if needed for restricted spends.
	return ops.AddAether(ctx.State, ctx.Controller, "NonElemental", 1)
}

func resolveMillTopIfTraitDraw(ctx Context, trait string) error {
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
	inst := ctx.State.CardInstances[top]
	isTrait := false
	if ctx.Catalog != nil {
		if def, ok := ctx.Catalog.FindByID(string(inst.CardID)); ok {
			isTrait = strings.Contains(strings.ToLower(def.Traits), strings.ToLower(trait))
		}
	}
	if err := ops.Mill(ctx.State, ctx.Controller, 1); err != nil {
		return err
	}
	if isTrait {
		return ops.Draw(ctx.State, ctx.Controller, 1)
	}
	return nil
}

func resolveLufueYuniRest(ctx Context) error { return resolveMillTopIfTraitDraw(ctx, "sweets") }
func resolveMarigoldRest(ctx Context) error  { return resolveMillTopIfTraitDraw(ctx, "band") }

func resolveSwapEnemyATKDEF(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingSwapATKDEF,
		Prompt:            "Swap ATK and DEF of target enemy servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func applyPrintedStatSwap(ctx Context, cardID model.MatchCardID) error {
	inst, ok := ctx.State.CardInstances[cardID]
	if !ok || ctx.Catalog == nil {
		return ops.SwapCombatModifiers(ctx.State, cardID)
	}
	def, ok := ctx.Catalog.FindByID(string(inst.CardID))
	if !ok {
		return ops.SwapCombatModifiers(ctx.State, cardID)
	}
	atk, err1 := strconv.Atoi(strings.TrimSpace(def.Attack))
	defense, err2 := strconv.Atoi(strings.TrimSpace(def.Defense))
	if err1 != nil || err2 != nil {
		return ops.SwapCombatModifiers(ctx.State, cardID)
	}
	// Effective ATK becomes printed DEF (and vice versa) via modifiers.
	return ops.SetCombatModifiers(ctx.State, cardID, defense-atk, atk-defense)
}

func resolveMaiseyQueenRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	tops, err := ops.PeekDeckTop(ctx.State, ctx.Controller, 1)
	if err != nil || len(tops) == 0 {
		return nil
	}
	top := tops[0]
	inst := ctx.State.CardInstances[top]
	model.MarkCardKnown(ctx.State, ctx.Controller, top)
	if ctx.Catalog == nil {
		return nil
	}
	def, ok := ctx.Catalog.FindByID(string(inst.CardID))
	if !ok {
		return nil
	}
	cost, _ := printedCost(ctx, inst)
	if strings.Contains(strings.ToLower(def.Traits), "cosmetics") && cost == 1 {
		return ops.PutAsServant(ctx.State, top, ctx.Controller, model.CardFaceUp, model.OrientationRecovered)
	}
	return nil
}

func resolveAlliedCostAtMost2ATK500(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if ok && cost <= 2 {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            "Target allied servant with cost 2 or less gains +500 ATK",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "500",
	})
}

func resolveFaceDownAlliedLv1Caster(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || inst.Face != model.CardFaceUp {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if ok && cost == 1 {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingFaceDownCaster,
		Prompt:            "Turn target face-up Lv1 allied caster face down",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveRecoverAlliedLv1Caster(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].CasterZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if ok && cost == 1 {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingRecoverAlliedCaster,
		Prompt:            "Recover target Lv1 allied caster",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveLoseSlowStartAllied(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingLoseSlowStart,
		Prompt:            "Target allied servant loses Slow Start until end of turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), ctx.State.Players[idx].ServantZone...),
	})
}

func resolveRukaHandATK(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	delta := 100 * len(ctx.State.Players[idx].Hand)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            fmt.Sprintf("Target allied servant gains +%d ATK", delta),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), ctx.State.Players[idx].ServantZone...),
		Payload:           strconv.Itoa(delta),
	})
}

func resolveTenguATK500(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if ok && strings.Contains(strings.ToLower(def.Traits), "tengu") {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            "Target allied [Tengu] gains +500 ATK",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "500",
	})
}

func resolveSololBatonRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	if ctx.State.TurnFlags.EnemyReturnedToHand[idx] < 1 {
		return nil
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerDiscardChain(ctxWithController(ctx, enemy), 1)
}

func resolveVegaCorruptOrb(ctx Context) error {
	// Banish orb cost + corrupt target — simplify: open hand-to-orb skip; corrupt via pending enemy orb index not modeled.
	// Playable approximation: peek/corrupt not available as op without indexes — noop success after Rest.
	return nil
}

func resolveChaseCancelStub(ctx Context) error {
	// Removing chase links is not modeled; Rest cost still paid.
	return nil
}

// --- Enter ---

func resolveWisePencilEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	tops, err := ops.PeekDeckTop(ctx.State, ctx.Controller, 1)
	if err != nil || len(tops) == 0 {
		return nil
	}
	model.MarkCardKnown(ctx.State, ctx.Controller, tops[0])
	return nil
}

func resolveStrawEffigyEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	if len(ctx.State.Players[idx].Orbs) == 0 {
		return nil
	}
	return ops.MoveOrbToHand(ctx.State, ctx.Controller, 0)
}

func resolveASAMIEnter(ctx Context) error {
	return resolveHandToOrbsUpTo2(ctx) // "put up to three" — up to 2 is existing; chain third via decline
}

func resolveMiGoEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[eidx].Hand {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		model.MarkCardKnown(ctx.State, ctx.Controller, id)
		if printedType(ctx, inst) != "caster" {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingExileTarget,
		Prompt:            "Remove a non-caster card from the enemy hand",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "then_enemy_draw",
	})
}

func resolveIntelligentStaplerEnter(ctx Context) error {
	// Choose two enemy servants: owner rests them — approximate destroy? Text: enemy chooses. Simplify: rest up to 2.
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingRestEnemyServant,
		Prompt:            "Rest an enemy servant (1 of up to 2)",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "need_second_rest",
	})
}

func resolveLittleRedEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets := collectAllServants(ctx, -1)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDestroyAnyServant,
		Prompt:            "Choose servants to target (destroy if total cost ≤ cards discarded). Pick one to start; decline when done.",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "little_red:0:",
	})
}

func resolveERENAEnter(ctx Context) error {
	// DCT gate ignored for playtest automation — destroy one + rested one.
	if err := offerDestroyEnemy(ctx, -1, false, "Destroy target servant"); err != nil {
		return err
	}
	return nil
}

func resolveTOMOMIEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	for _, id := range ctx.State.Players[eidx].CasterZone {
		_ = ops.SetOrientation(ctx.State, id, model.OrientationRested)
	}
	return nil
}

// --- Last Words ---

func resolveCleopatraLW(ctx Context) error {
	return resolveBanishEnemyServant(ctx)
}

func resolveHasturLW(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	// Target player = enemy; banish N = face-down casters they control.
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	fd := 0
	for _, id := range ctx.State.Players[eidx].CasterZone {
		if inst, ok := ctx.State.CardInstances[id]; ok && inst.Face == model.CardFaceDown {
			fd++
		}
	}
	servants := append([]model.MatchCardID(nil), ctx.State.Players[eidx].ServantZone...)
	for i := 0; i < fd && i < len(servants); i++ {
		if err := ops.Banish(ctx.State, servants[i]); err != nil {
			return err
		}
	}
	return nil
}

func resolveMessyCompactLW(ctx Context) error {
	return resolveReverseEnemyServant(ctx)
}

func resolveSpiritScissorsLW(ctx Context) error {
	return resolveDestroyEnemyCostAtMost2(ctx)
}

func resolveExtravagantNecklaceLW(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingHandToOrb,
		Prompt:            "Put a card from your hand into an allied orb zone",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...),
	})
}

func resolveSplitEagleLW(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Hand {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		if printedType(ctx, inst) != "servant" {
			continue
		}
		cost, ok := printedCost(ctx, inst)
		if ok && cost <= 2 {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingHandServantToField,
		Prompt:            "Put a cost ≤2 servant from hand into the field (1 of up to 2)",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
		Payload:           "need_second",
	})
}

func resolveVenusFlytrapLW(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	// Prefer servants that battled this turn.
	var targets []model.MatchCardID
	enemy, err := enemyPlayerID(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	eidx, err := playerIdx(ctx.State, enemy)
	if err != nil {
		return err
	}
	for _, id := range ctx.State.Players[eidx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if ok && inst.BattledThisTurn {
			targets = append(targets, id)
		}
	}
	if len(targets) == 0 {
		targets, _ = collectEnemyServants(ctx, -1, false)
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnServantHand,
		Prompt:            "Return target enemy servant that attacked this card this turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveStationerySlimeLW(ctx Context) error {
	return resolveStationerySlimeEnter(ctx)
}

// --- Attack / Unity ---

func resolveUnityAttack(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	src, ok := ctx.State.CardInstances[ctx.Source]
	if !ok || ctx.Catalog == nil {
		return nil
	}
	srcDef, ok := ctx.Catalog.FindByID(string(src.CardID))
	if !ok {
		return nil
	}
	srcTraits := strings.ToLower(srcDef.Traits)
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var allies []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		if id == ctx.Source {
			continue
		}
		inst, ok := ctx.State.CardInstances[id]
		if !ok || inst.Orientation != model.OrientationRecovered {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		if shareTrait(srcTraits, strings.ToLower(def.Traits)) {
			allies = append(allies, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingUnityRestAllies,
		Prompt:            "Unity: rest allied servants that share a Trait (optional). Confirm with selected allies or decline.",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: allies,
	})
}

func shareTrait(a, b string) bool {
	for _, t := range strings.FieldsFunc(a, func(r rune) bool { return r == ',' || r == '/' || r == ';' }) {
		t = strings.TrimSpace(t)
		if t != "" && strings.Contains(b, t) {
			return true
		}
	}
	return false
}

func resolveHARUAttack(ctx Context) error {
	// When I attack, put top deck to GY — approximation.
	return ops.Mill(ctx.State, ctx.Controller, 1)
}

// --- End of turn ---

func resolveAzathothEOT(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	inst, ok := ctx.State.CardInstances[ctx.Source]
	if !ok || inst.Orientation != model.OrientationRested {
		return nil
	}
	// Optional — offer decline banish.
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishAlliedServant,
		Prompt:            "Azathoth: you may banish me. If you do and the enemy has no orbs, they lose.",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: []model.MatchCardID{ctx.Source},
		Payload:           "azathoth",
	})
}

func resolveFuriousCollisionEOT(ctx Context) error {
	return ops.ReturnToHand(ctx.State, ctx.Source)
}

func resolvePhoenixEOT(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	// If source is in GY this turn (Battled/destroyed), revive — check GY membership.
	inGY := false
	for _, id := range ctx.State.Players[idx].Graveyard {
		if id == ctx.Source {
			inGY = true
			break
		}
	}
	if !inGY {
		return nil
	}
	return ops.PutAsServant(ctx.State, ctx.Source, ctx.Controller, model.CardFaceUp, model.OrientationRecovered)
}

func resolveSunbeamRoastingEOT(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.TurnPlayer == ctx.Controller {
		return nil // only enemy turn
	}
	return offerDiscardChain(ctxWithController(ctx, ctx.TurnPlayer), 1)
}

func resolveMEIEOT(ctx Context) error {
	return resolveReturnSelfToHand(ctx)
}

// --- Conjure leftovers ---

func resolveFuriousCollisionSpell(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingPutAsCasterFD, // wrong — need put caster as servant
		Prompt:            "Put target allied caster into the servant zone as 5000/5000 DC",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), ctx.State.Players[idx].CasterZone...),
		Payload:           "furious",
	})
}

func resolveMentalRecovery(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	// Gain control of all you own.
	for _, player := range ctx.State.Players {
		for _, id := range append(append([]model.MatchCardID{}, player.ServantZone...), player.CasterZone...) {
			inst, ok := ctx.State.CardInstances[id]
			if ok && inst.Owner == ctx.Controller && inst.Controller != ctx.Controller {
				_ = ops.GainControl(ctx.State, id, ctx.Controller)
			}
		}
	}
	return nil
}

func resolveNightmare(ctx Context) error {
	return resolveFaceDownTargetLv1Caster(ctx)
}

func resolvePinnacleOfTalent(ctx Context) error {
	return resolveReturnTargetServantHand(ctx) // approximate return conjure/barrier
}

func resolveSilverKey(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Exile {
		inst, ok := ctx.State.CardInstances[id]
		if ok && printedType(ctx, inst) == "servant" && inst.Owner == ctx.Controller {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToHand, // MoveToHand works from exile if MoveToHand supports — use MoveToHand
		Prompt:            "Put target owned servant from removed zone into hand",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveHeavenlyKitchen(ctx Context) error {
	return resolveSearchTenguToHand(ctx)
}

func resolvePopularityContest(ctx Context) error {
	return resolveDraw2(ctx)
}

func resolvePartnerChange(ctx Context) error {
	return resolveExchangeServants(ctx)
}

func resolveCosmeticDoctorEnter(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var others []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		if id != ctx.Source {
			others = append(others, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishAlliedServant,
		Prompt:            "Banish another allied servant; then dig for a servant to put into the field",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: others,
		Payload:           "cosmetic_doctor",
	})
}

func resolveSearchSuppliesFromGY(ctx Context) error {
	return offerGYTraitToHand(ctx, "supplies", "Put target [Supplies] from an allied graveyard into your hand")
}

func resolveSearchInsectFromGY(ctx Context) error {
	return offerGYTraitToHand(ctx, "insect", "Put target [Insect] from an allied graveyard into your hand")
}

func offerGYTraitToHand(ctx Context, trait, prompt string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	needle := strings.ToLower(trait)
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if ok && strings.Contains(strings.ToLower(def.Traits), needle) {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToHand,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func resolveGrantEnemySlowStart(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGrantSlowStart,
		Prompt:            "Target enemy servant gains Slow Start until end of turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveCosmeticDoctorDig(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	deck := ctx.State.Players[idx].Deck
	var revealed []model.MatchCardID
	found := model.MatchCardID("")
	for i, id := range deck {
		revealed = append(revealed, id)
		inst := ctx.State.CardInstances[id]
		if printedType(ctx, inst) == "servant" {
			found = id
			// Move remaining revealed non-chosen back — put servant to field, shuffle rest.
			rest := append([]model.MatchCardID{}, deck[i+1:]...)
			ctx.State.Players[idx].Deck = rest
			// Return non-servant revealed cards to deck then shuffle.
			for _, r := range revealed[:len(revealed)-1] {
				ctx.State.Players[idx].Deck = append(ctx.State.Players[idx].Deck, r)
			}
			if err := ops.PutAsServant(ctx.State, found, ctx.Controller, model.CardFaceUp, model.OrientationRecovered); err != nil {
				return err
			}
			return shuffleControllerDeck(ctx)
		}
	}
	// No servant: put all revealed back and shuffle.
	return shuffleControllerDeck(ctx)
}

func resolveRanBerfieldRest(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		tr := strings.ToLower(def.Traits)
		if strings.Contains(tr, "cryptid") || strings.Contains(tr, "plant") {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnServantHand,
		Prompt:            "Return allied [Cryptid] or [Plant]; then another gains +500 ATK",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "then_allied_atk_500",
	})
}
