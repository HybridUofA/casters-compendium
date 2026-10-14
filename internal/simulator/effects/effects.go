// Package effects holds typed card-effect hooks for the simulator.
//
// Hybrid owns Enter, Last Words, continuous ATK/DEF modifiers, and activated
// ability automation. Do not add a general card-text parser here.
//
// Card recipes live in master_effects.go (one table for the whole catalog).
// That mirrors EDOPro's "register effect + operation" model without one file
// per card — TCC's pool is small enough to audit in a single list. Fill
// EffectDecl.Resolve to automate; leave nil for manual play.
//
// Targeting uses Context.OfferChoice (engine injects OfferPendingChoice) so this
// package never imports engine.
package effects

import (
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// Hook identifies when an effect may fire.
type Hook string

const (
	HookEnter     Hook = "enter"
	HookLastWords Hook = "last_words"
	// HookRest is an activated Rest ability after its Rest cost is paid.
	HookRest Hook = "rest"
	// HookAttack fires when a Servant declares an attack (automatic "whenever I attack").
	HookAttack Hook = "attack"
	// HookEndOfTurn fires during End Phase for "at end of turn" automatics.
	HookEndOfTurn Hook = "end_of_turn"
	// HookResolve fires when a Conjure (one-shot spell) resolves from Chase.
	HookResolve Hook = "resolve"
)

// Catalog is the printed-card lookup used by cost/type filters in resolves.
type Catalog interface {
	FindByID(id string) (gamecards.Card, bool)
}

// Context is the Hybrid-owned environment for applying one effect trigger.
type Context struct {
	State      *model.MatchState
	Source     model.MatchCardID
	Controller model.PlayerID
	Hook       Hook
	Catalog    Catalog
	// TurnPlayer is the player whose turn is ending (HookEndOfTurn) or active.
	TurnPlayer model.PlayerID
	// OfferChoice opens a PendingChoice without importing engine.
	OfferChoice func(model.PendingChoice) error
	// DestroyField destroys a Servant-zone card and runs Last Words.
	DestroyField func(cardID model.MatchCardID) error
}

// Handler is one typed printed-ability pattern.
type Handler interface {
	Matches(hook Hook, cardID model.CardID, abilityText string) bool
	OnTrigger(ctx Context) error
}

// Registry stores Hybrid-owned handlers.
type Registry struct {
	handlers []Handler
}

// Register appends a handler.
func (r *Registry) Register(handler Handler) {
	if r == nil || handler == nil {
		return
	}
	r.handlers = append(r.handlers, handler)
}

// Handlers returns the registered handlers in registration order.
func (r *Registry) Handlers() []Handler {
	if r == nil {
		return nil
	}
	out := make([]Handler, len(r.handlers))
	copy(out, r.handlers)
	return out
}

// HasMatch reports whether any registered handler Matches the card for hook.
func HasMatch(registry *Registry, hook Hook, cardID model.CardID, abilityText string) bool {
	if registry == nil {
		return false
	}
	for _, handler := range registry.Handlers() {
		if handler != nil && handler.Matches(hook, cardID, abilityText) {
			return true
		}
	}
	return false
}

// ApplyHook runs every registered handler that Matches the printed card for ctx.Hook.
func ApplyHook(registry *Registry, ctx Context, cardID model.CardID, abilityText string) error {
	if registry == nil {
		return nil
	}
	for _, handler := range registry.Handlers() {
		if handler == nil || !handler.Matches(ctx.Hook, cardID, abilityText) {
			continue
		}
		if err := handler.OnTrigger(ctx); err != nil {
			return err
		}
	}
	return nil
}
