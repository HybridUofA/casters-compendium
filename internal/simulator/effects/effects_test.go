package effects

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type stubHandler struct {
	cardID model.CardID
	called bool
}

func (h *stubHandler) Matches(cardID model.CardID, _ string) bool {
	return cardID == h.cardID
}

func (h *stubHandler) OnTrigger(ctx Context) error {
	h.called = true
	_ = ctx
	return nil
}

func TestRegistryRegisterAndHandlers(t *testing.T) {
	var registry Registry
	handler := &stubHandler{cardID: "card-1"}
	registry.Register(handler)
	handlers := registry.Handlers()
	if len(handlers) != 1 {
		t.Fatalf("Handlers() len = %d; want 1", len(handlers))
	}
	if !handlers[0].Matches("card-1", "") {
		t.Fatal("registered handler did not match")
	}
	if err := handlers[0].OnTrigger(Context{Hook: HookEnter}); err != nil {
		t.Fatalf("OnTrigger() error = %v", err)
	}
	if !handler.called {
		t.Fatal("expected OnTrigger to mark handler called")
	}
}
