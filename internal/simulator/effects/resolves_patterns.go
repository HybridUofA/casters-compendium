package effects

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// resolveNoop marks keyword-only / continuous reminders handled elsewhere.
func resolveNoop(Context) error { return nil }

func resolvePriceOfOvereating(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if err := ops.Draw(ctx.State, ctx.Controller, 4); err != nil {
		return err
	}
	return offerDiscardChain(ctx, 4)
}

func offerDiscardChain(ctx Context, remaining int) error {
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	if remaining <= 0 || len(hand) == 0 {
		return nil
	}
	payload := fmt.Sprintf("discard_left:%d", remaining-1)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDiscardOneFromHand,
		Prompt:            fmt.Sprintf("Discard a card (%d remaining after this)", remaining-1),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: hand,
		Payload:           payload,
	})
}

func resolveReturnTargetServantHand(ctx Context) error {
	return offerReturnServants(ctx, -1, "Return target servant to its owner's hand")
}

func resolveReturnTargetServantHandCostAtMost4(ctx Context) error {
	return offerReturnServants(ctx, 4, "Return target servant with cost 4 or less to its owner's hand")
}

func resolveReturnTwoTargetServants(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets := collectAllServants(ctx, -1)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnTwoServants,
		Prompt:            "Return a servant to its owner's hand (1 of 2)",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "need_second",
	})
}

func offerReturnServants(ctx Context, maxCost int, prompt string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnServantHand,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: collectAllServants(ctx, maxCost),
	})
}

func collectAllServants(ctx Context, maxCost int) []model.MatchCardID {
	var out []model.MatchCardID
	for _, player := range ctx.State.Players {
		for _, id := range player.ServantZone {
			inst, ok := ctx.State.CardInstances[id]
			if !ok || !isServantInstance(ctx, inst) {
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
	}
	return out
}

func resolveGainControlEnemyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGainControlServant,
		Prompt:            "Gain control of target enemy servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveRestOrReverseTarget(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets := collectAllServants(ctx, -1)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingOrientRestOrReverse,
		Prompt:            "Choose a servant, then Rest or Reverse",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Options: []model.PendingChoiceOption{
			{ID: "rest", Label: "Rest"},
			{ID: "reverse", Label: "Reverse"},
		},
	})
}

func resolveRestOrReverseEnemy(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingOrientRestOrReverse,
		Prompt:            "Choose an enemy servant, then Rest or Reverse",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Options: []model.PendingChoiceOption{
			{ID: "rest", Label: "Rest"},
			{ID: "reverse", Label: "Reverse"},
		},
	})
}

func resolveDestroyTargetBarrierSpell(ctx Context) error {
	return resolveDestroyTargetBarrier(ctx)
}

func resolveExileEnemyServantCostAtMost3(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, 3, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingExileTarget,
		Prompt:            "Remove target enemy servant with cost 3 or less from the game",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveBanishEnemyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishEnemyServant,
		Prompt:            "Target enemy player banishes a servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveSearchServantToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a servant", func(defName, defType, traits string, cost int) bool {
		return strings.EqualFold(defType, "Servant")
	})
}

func resolveSearchLv1CasterToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a Lv1 caster", func(_, defType, _ string, cost int) bool {
		return strings.EqualFold(defType, "Caster") && cost == 1
	})
}

func resolveSearchLv2CasterToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a Lv2 caster", func(_, defType, _ string, cost int) bool {
		return strings.EqualFold(defType, "Caster") && cost == 2
	})
}

func resolveSearchJewelryToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a [Jewelry]", func(_, _, traits string, _ int) bool {
		return strings.Contains(strings.ToLower(traits), "jewelry")
	})
}

func resolveSearchSuppliesToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a [Supplies]", func(_, _, traits string, _ int) bool {
		return strings.Contains(strings.ToLower(traits), "supplies")
	})
}

func resolveSearchYoukaiCost1ToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a [Youkai] with cost 1", func(_, _, traits string, cost int) bool {
		return strings.Contains(strings.ToLower(traits), "youkai") && cost == 1
	})
}

func resolveSearchServantCost7PlusToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a servant with cost 7 or more", func(_, defType, _ string, cost int) bool {
		return strings.EqualFold(defType, "Servant") && cost >= 7
	})
}

func resolveSearchTenguToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a [Tengu]", func(_, _, traits string, _ int) bool {
		return strings.Contains(strings.ToLower(traits), "tengu")
	})
}

func resolveSearchSweetsCost1ToHand(ctx Context) error {
	return offerSearchDeck(ctx, PendingSearchDeckToHand, "Search your deck for a Sweets with cost 1", func(_, _, traits string, cost int) bool {
		return strings.Contains(strings.ToLower(traits), "sweets") && cost == 1
	})
}

type deckFilter func(name, typ, traits string, cost int) bool

func offerSearchDeck(ctx Context, kind string, prompt string, filter deckFilter) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Deck {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		cost, _ := strconvAtoi(def.CostLevel)
		if filter(def.Name, def.Type, def.Traits, cost) {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              kind,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func strconvAtoi(s string) (int, error) {
	n := 0
	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not an int")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func resolveGYServantCostAtMost3ToHand(ctx Context) error {
	return offerGY(ctx, PendingGYToHand, 3, "servant", "Put target servant with cost 3 or less from an allied graveyard into your hand")
}

func resolveGYSuppliesToDeckTop(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(def.Traits), "supplies") {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToDeckTop,
		Prompt:            "Put target [Supplies] from an allied graveyard on top of your deck",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func resolveGYSweetsCost1ToField(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		cost, _ := strconvAtoi(def.CostLevel)
		if strings.Contains(strings.ToLower(def.Traits), "sweets") && cost == 1 {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToField,
		Prompt:            "Put target [Sweets] with cost 1 from an allied graveyard into the field",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func offerGY(ctx Context, kind string, maxCost int, typ string, prompt string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var matches []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		if typ != "" && printedType(ctx, inst) != "" && printedType(ctx, inst) != typ {
			continue
		}
		if maxCost >= 0 {
			cost, ok := printedCost(ctx, inst)
			if !ok || cost > maxCost {
				continue
			}
		}
		matches = append(matches, id)
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              kind,
		Prompt:            prompt,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func resolveHandToOrbsUpTo2(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	hand := append([]model.MatchCardID(nil), ctx.State.Players[idx].Hand...)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingHandToOrbUpTo2,
		Prompt:            "Put a hand card into an orb zone (up to two)",
		AllowDecline:      true,
		SourceCardID:      ctx.Source,
		SelectableCardIDs: hand,
	})
}

func resolvePeekEnemyOrb(ctx Context) error {
	return offerPeekEnemyOrb(ctx, "then_draw")
}

func resolvePeekEnemyOrbOnly(ctx Context) error {
	return offerPeekEnemyOrb(ctx, "")
}

func offerPeekEnemyOrb(ctx Context, payload string) error {
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
	orbs := append([]model.MatchCardID(nil), ctx.State.Players[idx].Orbs...)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingPeekOrb,
		Prompt:            "Look at target enemy orb",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: orbs,
		Payload:           payload,
	})
}

func resolveAlliedATK2000(ctx Context) error {
	return resolveAlliedATK(2000)(ctx)
}

func resolveAlliedATK1000(ctx Context) error {
	return resolveAlliedATK(1000)(ctx)
}

func resolveAlliedATK500(ctx Context) error {
	return resolveAlliedATK(500)(ctx)
}

func resolveAlliedATK(delta int) func(Context) error {
	return func(ctx Context) error {
		if err := requireState(ctx); err != nil {
			return err
		}
		idx, err := playerIdx(ctx.State, ctx.Controller)
		if err != nil {
			return err
		}
		targets := zoneServants(ctx, idx)
		return offerOrSkip(ctx, model.PendingChoice{
			PlayerID:          ctx.Controller,
			Kind:              PendingModifyATK,
			Prompt:            fmt.Sprintf("Target allied servant gains %+d ATK until end of turn", delta),
			SourceCardID:      ctx.Source,
			SelectableCardIDs: targets,
			Payload:           fmt.Sprintf("%d", delta),
		})
	}
}

func resolveAlliedATK4000AndDC(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	targets := zoneServants(ctx, idx)
	// Use ATK pending; grant DC in continuation via combined kind — do ATK then DC on same target.
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            "Target allied servant gains +4000 ATK and Double Corrupt until end of turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "4000+dc",
	})
}

func resolvePutServantAsCasterFD(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingPutAsCasterFD,
		Prompt:            "Put target servant into its owner's caster zone face down",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: collectAllServants(ctx, -1),
	})
}

func resolveFaceDownTargetLv1Caster(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, player := range ctx.State.Players {
		for _, id := range player.CasterZone {
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
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingFaceDownCaster,
		Prompt:            "Turn target Lv1 caster face down",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveBanishSelf(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.Banish(ctx.State, ctx.Source)
}

func resolveBottomSelf(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.ReturnToDeck(ctx.State, ctx.Source, model.DeckPlacementBottom)
}

func resolveDestroySelf(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.DestroyField == nil {
		return fmt.Errorf("DestroyField required")
	}
	return ctx.DestroyField(ctx.Source)
}

func resolveReturnSelfToHand(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	return ops.ReturnToHand(ctx.State, ctx.Source)
}

func resolveStationerySlimeEnter(ctx Context) error {
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
	putField := false
	if ctx.Catalog != nil {
		if def, ok := ctx.Catalog.FindByID(string(inst.CardID)); ok {
			cost, _ := strconvAtoi(def.CostLevel)
			if strings.Contains(strings.ToLower(def.Traits), "supplies") &&
				strings.EqualFold(def.Type, "Servant") && cost <= 4 {
				putField = true
			}
		}
	}
	if putField {
		return ops.PutAsServant(ctx.State, top, ctx.Controller, model.CardFaceUp, model.OrientationRecovered)
	}
	return ops.MoveFromDeckToHand(ctx.State, ctx.Controller, top)
}

func resolveCatGirlEnter(ctx Context) error {
	// Reveal until Demon/Cat/Youkai cost≤2 — dig top until match.
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	deck := append([]model.MatchCardID(nil), ctx.State.Players[idx].Deck...)
	var revealed []model.MatchCardID
	var found model.MatchCardID
	for _, id := range deck {
		revealed = append(revealed, id)
		inst := ctx.State.CardInstances[id]
		if ctx.Catalog != nil {
			if def, ok := ctx.Catalog.FindByID(string(inst.CardID)); ok {
				cost, _ := strconvAtoi(def.CostLevel)
				traits := strings.ToLower(def.Traits)
				if strings.EqualFold(def.Type, "Servant") && cost <= 2 &&
					(strings.Contains(traits, "demon") || strings.Contains(traits, "cat") || strings.Contains(traits, "youkai")) {
					found = id
					break
				}
			}
		}
	}
	for _, id := range revealed {
		if id == found {
			continue
		}
		if err := moveDeckCardToGY(ctx, id); err != nil {
			return err
		}
	}
	if found != "" {
		return ops.PutAsServant(ctx.State, found, ctx.Controller, model.CardFaceUp, model.OrientationRecovered)
	}
	return nil
}

func moveDeckCardToGY(ctx Context, cardID model.MatchCardID) error {
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	deck := ctx.State.Players[idx].Deck
	for i, id := range deck {
		if id != cardID {
			continue
		}
		ctx.State.Players[idx].Deck = append(append([]model.MatchCardID{}, deck[:i]...), deck[i+1:]...)
		inst := ctx.State.CardInstances[cardID]
		clear := inst
		clear.Controller = clear.Owner
		ctx.State.CardInstances[cardID] = clear
		ctx.State.Players[idx].Graveyard = append(ctx.State.Players[idx].Graveyard, cardID)
		return nil
	}
	return fmt.Errorf("card %q not in deck", cardID)
}
