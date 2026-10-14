// Package effects holds typed card-effect hooks for the simulator.
//
// Hybrid owns Enter, Last Words, continuous ATK/DEF modifiers, and activated
// ability automation. Do not add a general card-text parser here; implement
// typed handlers for a small card pool and generalize after patterns repeat.
//
// UI and session layers already present model.PendingChoice prompts. Effect
// handlers should open choices via engine.OfferPendingChoice and resolve
// Kind-specific outcomes when Hybrid wires ResolvePendingChoice side effects.
package effects

import "github.com/HybridUofA/casters-compendium/internal/simulator/model"

// Hook identifies when an effect may fire.
type Hook string

const (
	HookEnter     Hook = "enter"
	HookLastWords Hook = "last_words"
)

// Context is the Hybrid-owned environment for applying one effect trigger.
type Context struct {
	State      *model.MatchState
	Source     model.MatchCardID
	Controller model.PlayerID
	Hook       Hook
}

// Handler is one typed printed-ability pattern. Hybrid registers handlers and
// invokes them from engine resolution points (servant enters field, leaves to
// graveyard, continuous stat reads, etc.).
type Handler interface {
	// Matches reports whether this handler applies to the printed definition.
	Matches(cardID model.CardID, abilityText string) bool
	// OnTrigger may mutate state and/or open a PendingChoice.
	OnTrigger(ctx Context) error
}

// Registry stores Hybrid-owned handlers. Empty until Hybrid registers cards.
type Registry struct {
	handlers []Handler
}

// Register appends a handler. Safe for Hybrid to call during engine setup.
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
