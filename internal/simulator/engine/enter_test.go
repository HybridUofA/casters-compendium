package engine

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type enterStubHandler struct {
	cardID model.CardID
	called bool
	source model.MatchCardID
}

func (h *enterStubHandler) Matches(hook effects.Hook, cardID model.CardID, _ string) bool {
	return hook == effects.HookEnter && cardID == h.cardID
}

func (h *enterStubHandler) OnTrigger(ctx effects.Context) error {
	h.called = true
	h.source = ctx.Source
	return nil
}

func TestTriggerEnterEffectsRunsRegisteredHandler(t *testing.T) {
	t.Cleanup(ResetEffectHandlers)
	handler := &enterStubHandler{cardID: "printed-servant"}
	RegisterEffectHandler(handler)

	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant-1": {
				MatchID: "servant-1", CardID: "printed-servant",
				Owner: "player-one", Controller: "player-one",
			},
		},
	}
	if err := triggerEnterEffects(&state, nil, "servant-1"); err != nil {
		t.Fatalf("triggerEnterEffects() error = %v", err)
	}
	if !handler.called || handler.source != "servant-1" {
		t.Fatalf("handler called=%v source=%q", handler.called, handler.source)
	}
}

func TestTriggerEnterEffectsNoHandlersIsNoOp(t *testing.T) {
	t.Cleanup(ResetEffectHandlers)
	state := model.MatchState{
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"servant-1": {
				MatchID: "servant-1", CardID: "printed-servant",
				Owner: "player-one", Controller: "player-one",
			},
		},
	}
	if err := triggerEnterEffects(&state, nil, "servant-1"); err != nil {
		t.Fatalf("triggerEnterEffects() error = %v", err)
	}
}
