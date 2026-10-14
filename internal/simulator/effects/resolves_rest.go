package effects

import (
	"fmt"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

const PendingDiscardCasterThen = "fx:discard_caster_then"

func resolveAlliedDEF1000(ctx Context) error {
	return offerModifyDEF(ctx, 1000, true)
}

func resolveAlliedDEF500(ctx Context) error {
	return offerModifyDEF(ctx, 500, true)
}

func resolveRestDiscardCasterThenDraw(ctx Context) error {
	return offerDiscardCasterThen(ctx, "draw_1")
}

func resolveRestBanishTraitThenDraw(ctx Context) error {
	// Almeria-style: banish a [Cosmetics] from field/hand — treat as hand/field banish then draw.
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var candidates []model.MatchCardID
	for _, id := range ctx.State.Players[idx].ServantZone {
		inst, ok := ctx.State.CardInstances[id]
		if !ok || ctx.Catalog == nil {
			continue
		}
		def, ok := ctx.Catalog.FindByID(string(inst.CardID))
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(def.Traits), "cosmetics") {
			candidates = append(candidates, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishAlliedServant, // reuse banish-any
		Prompt:            "Banish a [Cosmetics], then draw",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: candidates,
		Payload:           "then_draw",
	})
}

func resolveEnemyATKMinus1000(ctx Context) error {
	return offerModifyATK(ctx, -1000, false)
}

func resolveEnemyATKMinus500(ctx Context) error {
	return offerModifyATK(ctx, -500, false)
}

func offerModifyATK(ctx Context, delta int, allied bool) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	var targets []model.MatchCardID
	if allied {
		idx, err := playerIdx(ctx.State, ctx.Controller)
		if err != nil {
			return err
		}
		targets = append(targets, ctx.State.Players[idx].ServantZone...)
	} else {
		var err error
		targets, err = collectEnemyServants(ctx, -1, false)
		if err != nil {
			return err
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyATK,
		Prompt:            fmt.Sprintf("Choose a servant (%+d ATK until end of turn)", delta),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           fmt.Sprintf("%d", delta),
	})
}

func offerModifyDEF(ctx Context, delta int, allied bool) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	targets := append([]model.MatchCardID(nil), ctx.State.Players[idx].ServantZone...)
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingModifyDEF,
		Prompt:            fmt.Sprintf("Choose an allied servant (%+d DEF until end of turn)", delta),
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
		Payload:           fmt.Sprintf("%d", delta),
	})
}

func resolveGrantAlliedDC(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingGrantDC,
		Prompt:            "Target allied servant gains Double Corrupt until end of turn",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), ctx.State.Players[idx].ServantZone...),
	})
}

func resolveReverseEnemyServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, -1, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingOrientReverse,
		Prompt:            "Reverse target enemy servant",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

func resolveDestroyEnemyCostAtMost3(ctx Context) error {
	return resolveDestroyEnemyServantCostAtMost3(ctx)
}

func resolveReturnEnemyCostAtMost4(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	targets, err := collectEnemyServants(ctx, 4, false)
	if err != nil {
		return err
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingReturnServantHand,
		Prompt:            "Return target enemy servant with cost 4 or less to hand",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: targets,
	})
}

// Rest cost: discard a caster, then run a follow-up via Payload.
func resolveRestDiscardCasterThenDestroy3(ctx Context) error {
	return offerDiscardCasterThen(ctx, "destroy_enemy_3")
}

func resolveRestDiscardCasterThenReverse(ctx Context) error {
	return offerDiscardCasterThen(ctx, "reverse_enemy")
}

func resolveRestDiscardCasterThenGrantDC(ctx Context) error {
	return offerDiscardCasterThen(ctx, "grant_dc")
}

func resolveRestDiscardCasterThenATK1000(ctx Context) error {
	return offerDiscardCasterThen(ctx, "atk_1000")
}

func offerDiscardCasterThen(ctx Context, then string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	idx, err := playerIdx(ctx.State, ctx.Controller)
	if err != nil {
		return err
	}
	var casters []model.MatchCardID
	for _, id := range ctx.State.Players[idx].Hand {
		inst, ok := ctx.State.CardInstances[id]
		if !ok {
			continue
		}
		if printedType(ctx, inst) == "caster" {
			casters = append(casters, id)
		}
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingDiscardCasterThen,
		Prompt:            "Discard a caster",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: casters,
		Payload:           then,
	})
}

func resolveAttackDestroyTargetServant(ctx Context) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	if ctx.State.Attack.TargetKind != model.AttackTargetServant || ctx.State.Attack.TargetCardID == "" {
		return nil
	}
	if ctx.DestroyField == nil {
		return fmt.Errorf("DestroyField required")
	}
	return ctx.DestroyField(ctx.State.Attack.TargetCardID)
}

const PendingBanishStockThen = "fx:banish_stock_then"

func resolveRestBanishStockThenDestroy3(ctx Context) error {
	return offerBanishStockThen(ctx, "destroy_enemy_3")
}

func resolveRestBanishStockThenATK1000(ctx Context) error {
	return offerBanishStockThen(ctx, "atk_1000")
}

func resolveRestBanishStockThenDraw(ctx Context) error {
	return offerBanishStockThen(ctx, "draw_1")
}

func offerBanishStockThen(ctx Context, then string) error {
	if err := requireState(ctx); err != nil {
		return err
	}
	host, ok := ctx.State.CardInstances[ctx.Source]
	if !ok || len(host.Stock) == 0 {
		return nil
	}
	return offerOrSkip(ctx, model.PendingChoice{
		PlayerID:          ctx.Controller,
		Kind:              PendingBanishStockThen,
		Prompt:            "Banish a card from my Stock",
		SourceCardID:      ctx.Source,
		SelectableCardIDs: append([]model.MatchCardID(nil), host.Stock...),
		Payload:           then,
	})
}
