package effects

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine/ops"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

const (
	PendingGrantVoidRest     = "fx:grant_void_rest"
	PendingDiscardThenGYName = "fx:discard_then_gy_name"
	PendingVoidServantATK500 = "fx:void_servant_atk_500"
)

// ResolveGrantedVoidRest runs the temporary Rest granted by Arthur.
func ResolveGrantedVoidRest(ctx Context) error {
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
		if strings.Contains(strings.ToLower(def.Element), "void") ||
			strings.EqualFold(strings.TrimSpace(def.Element), "Void") {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingVoidServantATK500,
		Prompt:            "Target allied {Void} servant gains +500 ATK until end of turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveArthurGrantVoidRest(ctx Context) error {
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
		if !ok || inst.Face != model.CardFaceDown {
			continue
		}
		targets = append(targets, id)
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGrantVoidRest,
		Prompt:            "Target allied face-down Caster gains Rest: Void servant +500 ATK this turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveArthurStockTop2Casters(ctx Context) error {
	return offerBanishStockThen(ctx, "arthur_top2_casters")
}

func resolveGretelRest(ctx Context) error {
	return offerDiscardThenGYNamed(ctx, "Hansel")
}

func resolveHanselRest(ctx Context) error {
	return offerDiscardThenGYNamed(ctx, "Gretel")
}

func offerDiscardThenGYNamed(ctx Context, name string) error {
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
		Kind:              PendingDiscardThenGYName,
		Prompt:            fmt.Sprintf("Discard a card, then put \"%s\" from GY onto the field", name),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: hand,
		Payload:           name,
	})
}

func resolveMariaAiDrawIdol(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.State.TurnFlags.IdolNonAetherAbility < 1 {
		return nil
	}
	return ops.Draw(ctx.State, ctx.Controller, 1)
}

func resolveMariaAiSearchBand(ctx Context) error {
	return offerBanishStockThen(ctx, "maria_search_band")
}

func resolveRanBerfieldExileEOT(ctx Context) error {
	return offerBanishStockThen(ctx, "ran_exile_eot")
}

func resolveGlowDesireATK2000(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	for _, id := range collectAllServants(ctx, -1) {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		printed, err := strconvAtoi(strings.TrimSpace(def.Attack))
		if err != nil {
			continue
		}
		if printed+inst.AttackModifier > printed {
			targets = append(targets, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            "Target servant with ATK > printed gains +2000 ATK",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "2000",
	})
}

func continueUniquePending(
	pending model.PendingChoice,
	selected []model.MatchCardID,
	ctx Context,
) (handled bool, err error) {
	switch pending.Kind {
	case PendingGrantVoidRest:
		if len(selected) != 1 {
			return true, fmt.Errorf("grant void rest requires one caster")
		}
		inst := ctx.State.CardInstances[selected[0]]
		inst.GrantedVoidRestATK500UntilEOT = true
		ctx.State.CardInstances[selected[0]] = inst
		return true, nil
	case PendingVoidServantATK500:
		if len(selected) != 1 {
			return true, fmt.Errorf("void ATK requires one servant")
		}
		return true, ops.ModifyAttack(ctx.State, selected[0], 500)
	case PendingDiscardThenGYName:
		if len(selected) != 1 {
			return true, fmt.Errorf("discard requires one card")
		}
		if err := ops.Discard(ctx.State, selected[0]); err != nil {
			return true, err
		}
		return true, offerGYNamedToField(ctx, pending.Payload)
	default:
		return false, nil
	}
}

func offerGYNamedToField(ctx Context, name string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	needle := strings.ToLower(strings.TrimSpace(name))
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
		if strings.EqualFold(strings.TrimSpace(def.Name), needle) {
			matches = append(matches, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGYToField,
		Prompt:            fmt.Sprintf("Put target \"%s\" from an allied graveyard onto the field", name),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: matches,
	})
}

func applyArthurTop2Casters(ctx Context) error {
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	n := 2
	if len(ctx.State.Players[idx].Deck) < n {
		n = len(ctx.State.Players[idx].Deck)
	}
	for i := 0; i < n; i++ {
		if len(ctx.State.Players[idx].Deck) == 0 {
			break
		}
		top := ctx.State.Players[idx].Deck[0]
		if err := ops.PutAsCaster(ctx.State, top, ctx.Controller, model.CardFaceDown, model.OrientationRested); err != nil {
			return err
		}
	}
	return nil
}

func applyMariaSearchBand(ctx Context) error {
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	earnest := 0
	for _, id := range ctx.State.Players[idx].Graveyard {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if ok && strings.Contains(strings.ToLower(def.Traits), "earnest") {
			earnest++
		}
	}
	return offerSearchDeck(ctx, PendingSearchDeckToField,
		fmt.Sprintf("Search your deck for a [Band] with cost ≤ %d", earnest),
		func(_, _, traits string, cost int) bool {
			return strings.Contains(strings.ToLower(traits), "band") && cost <= earnest
		})
}

func applyRanExileEOT(ctx Context) error {
	targets := collectAllServants(ctx, -1)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingExileTarget,
		Prompt:            "Remove target servant from the game (returns at end of turn)",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           "return_eot",
	})
}
