package effects

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type stubHandler struct {
	cardID model.CardID
	hook   Hook
	called bool
}

func (h *stubHandler) Matches(hook Hook, cardID model.CardID, _ string) bool {
	return hook == h.hook && cardID == h.cardID
}

func (h *stubHandler) OnTrigger(ctx Context) error {
	h.called = true
	_ = ctx
	return nil
}

func TestRegistryRegisterAndHandlers(t *testing.T) {
	var registry Registry
	handler := &stubHandler{cardID: "card-1", hook: HookEnter}
	registry.Register(handler)
	handlers := registry.Handlers()
	if len(handlers) != 1 {
		t.Fatalf("Handlers() len = %d; want 1", len(handlers))
	}
	if !handlers[0].Matches(HookEnter, "card-1", "") {
		t.Fatal("registered handler did not match")
	}
	if err := handlers[0].OnTrigger(Context{Hook: HookEnter}); err != nil {
		t.Fatalf("OnTrigger() error = %v", err)
	}
	if !handler.called {
		t.Fatal("expected OnTrigger to mark handler called")
	}
}

func TestApplyHookRunsMatchingHandlers(t *testing.T) {
	var registry Registry
	matched := &stubHandler{cardID: "card-1", hook: HookEnter}
	other := &stubHandler{cardID: "card-2", hook: HookEnter}
	wrongHook := &stubHandler{cardID: "card-1", hook: HookLastWords}
	registry.Register(matched)
	registry.Register(other)
	registry.Register(wrongHook)
	if err := ApplyHook(&registry, Context{Hook: HookEnter}, "card-1", ""); err != nil {
		t.Fatalf("ApplyHook() error = %v", err)
	}
	if !matched.called {
		t.Fatal("matching handler was not triggered")
	}
	if other.called {
		t.Fatal("non-matching handler was triggered")
	}
	if wrongHook.called {
		t.Fatal("wrong-hook handler was triggered")
	}
	if err := ApplyHook(nil, Context{Hook: HookEnter}, "card-1", ""); err != nil {
		t.Fatalf("ApplyHook(nil) error = %v", err)
	}
}

func TestHasMatch(t *testing.T) {
	var registry Registry
	registry.Register(&stubHandler{cardID: "card-1", hook: HookRest})
	if !HasMatch(&registry, HookRest, "card-1", "") {
		t.Fatal("expected HasMatch true")
	}
	if HasMatch(&registry, HookEnter, "card-1", "") {
		t.Fatal("expected HasMatch false for other hook")
	}
}
