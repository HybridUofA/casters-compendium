package engine

import (
	"fmt"
	"sync"

	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

// effectRegistry holds Hybrid-registered Enter / Last Words / Rest handlers.
// Empty by default; use RegisterDefaultEffectHandlers or RegisterEffectHandler.
var (
	effectRegistry      effects.Registry
	defaultHandlersOnce sync.Once
)

// RegisterEffectHandler appends a typed effect handler for chase/field triggers.
func RegisterEffectHandler(handler effects.Handler) {
	effectRegistry.Register(handler)
}

// RegisterDefaultEffectHandlers installs the shipped typed card handlers once.
func RegisterDefaultEffectHandlers() {
	defaultHandlersOnce.Do(func() {
		for _, handler := range effects.DefaultHandlers() {
			RegisterEffectHandler(handler)
		}
	})
}

// ResetEffectHandlers clears registered handlers. Intended for tests.
// Also allows RegisterDefaultEffectHandlers to run again.
func ResetEffectHandlers() {
	effectRegistry = effects.Registry{}
	defaultHandlersOnce = sync.Once{}
}

func triggerEnterEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookEnter, "")
}

func triggerLastWordsEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookLastWords, "")
}

func triggerRestEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookRest, "")
}

func triggerAttackEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookAttack, "")
}

func triggerResolveEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookResolve, "")
}

func triggerEndOfTurnEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
	turnPlayer model.PlayerID,
) error {
	return triggerHookEffects(state, catalog, sourceID, effects.HookEndOfTurn, turnPlayer)
}

// triggerFieldEndOfTurnEffects runs HookEndOfTurn for every card on the field.
func triggerFieldEndOfTurnEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	turnPlayer model.PlayerID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	var sources []model.MatchCardID
	for _, player := range state.Players {
		sources = append(sources, player.CasterZone...)
		sources = append(sources, player.ServantZone...)
	}
	for _, sourceID := range sources {
		if _, ok := state.CardInstances[sourceID]; !ok {
			continue
		}
		if err := triggerEndOfTurnEffects(state, catalog, sourceID, turnPlayer); err != nil {
			return err
		}
	}
	return nil
}

func triggerHookEffects(
	state *model.MatchState,
	catalog rules.CardCatalog,
	sourceID model.MatchCardID,
	hook effects.Hook,
	turnPlayer model.PlayerID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	instance, ok := state.CardInstances[sourceID]
	if !ok {
		return fmt.Errorf("effect source %q does not exist", sourceID)
	}
	abilityText := ""
	if catalog != nil {
		if definition, found := catalog.FindByID(string(instance.CardID)); found {
			abilityText = definition.Ability
		}
	}
	if turnPlayer == "" {
		turnPlayer = state.Turn.ActivePlayer
	}
	return effects.ApplyHook(
		&effectRegistry,
		effects.Context{
			State:      state,
			Source:     sourceID,
			Controller: instance.Controller,
			Hook:       hook,
			Catalog:    catalog,
			TurnPlayer: turnPlayer,
			OfferChoice: func(choice model.PendingChoice) error {
				return OfferPendingChoice(state, choice)
			},
			DestroyField: func(cardID model.MatchCardID) error {
				return destroyServantWithCause(state, catalog, cardID, destroyCauseEffect, instance.Controller)
			},
		},
		instance.CardID,
		abilityText,
	)
}
