package ai_test

import (
	"path/filepath"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/ai"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

func TestDriverPassAgentRunUntilIdleDoesNotHang(t *testing.T) {
	repository := tinyAIRepository(t)
	deck := tinyAIDeck(t, repository)
	engine.RegisterDefaultEffectHandlers()
	state, err := engine.BeginSetup(engine.SetupInput{
		Players: [2]engine.PlayerSetup{
			{ID: "player-one", Deck: deck},
			{ID: "player-two", Deck: deck},
		},
		Random:  engine.NewSeededRandom(engine.MatchSeed{First: 7, Second: 11}),
		Catalog: repository,
	})
	if err != nil {
		t.Fatalf("BeginSetup() error = %v", err)
	}
	match, err := session.NewLocalMatch(state, engine.MatchSeed{First: 7, Second: 11}, repository)
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	p1, err := session.NewPlayerSession(match, "player-one")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := session.NewPlayerSession(match, "player-two")
	if err != nil {
		t.Fatal(err)
	}

	// Finalize opening hands so the match is In Progress.
	v1, err := p1.View()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p1.SubmitOpeningHandDecision(nil, v1.Revision); err != nil {
		t.Fatal(err)
	}
	v2, err := p2.View()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p2.SubmitOpeningHandDecision(nil, v2.Revision); err != nil {
		t.Fatal(err)
	}

	driver := &ai.Driver{Session: p2, Agent: ai.PassAgent{}}

	// Drive P2 whenever it must act; advance P1 otherwise. Bounded so a hang fails.
	for step := 0; step < 80; step++ {
		before, err := p2.View()
		if err != nil {
			t.Fatal(err)
		}
		if before.MatchStatus == model.StatusFinished {
			return
		}

		_, cont, stepErr := driver.Step()
		if stepErr != nil {
			t.Fatalf("Step() error = %v at %d (phase=%s active=%s pri=%s open=%v)",
				stepErr, step, before.Turn.Phase, before.Turn.ActivePlayer,
				before.PriorityHolder, before.PrioritySequenceOpen)
		}
		if cont {
			continue
		}

		// P2 idle: nudge P1 so the match progresses through Main→Battle→End.
		p1View, viewErr := p1.View()
		if viewErr != nil {
			t.Fatal(viewErr)
		}
		advanced := false
		if p1View.PrioritySequenceOpen && p1View.PriorityHolder == "player-one" {
			if _, err = p1.PassPriority(p1View.Revision); err == nil {
				advanced = true
			} else if _, err = p1.CompleteCurrentPhase(p1View.Revision); err == nil {
				advanced = true
			}
		} else if p1View.Turn.ActivePlayer == "player-one" {
			if _, err = p1.CompleteCurrentPhase(p1View.Revision); err == nil {
				advanced = true
			} else if p1View.PrioritySequenceOpen {
				if _, err = p1.PassPriority(p1View.Revision); err == nil {
					advanced = true
				}
			}
		}
		if !advanced {
			// Reached a stable hand-off where neither seat can cheaply advance —
			// that still proves the driver stopped without looping forever.
			after, _ := p2.View()
			if after.PriorityHolder != "player-two" || !after.PrioritySequenceOpen {
				return
			}
			t.Fatalf("stuck: active=%s phase=%s pri=%s open=%v",
				after.Turn.ActivePlayer, after.Turn.Phase, after.PriorityHolder, after.PrioritySequenceOpen)
		}
		if step >= 20 {
			// Enough Main/Battle/End cycling with PassAgent to satisfy the plan.
			return
		}
	}
	t.Fatal("exceeded step budget")
}

func TestNaivePaymentBuildsIgnusCost(t *testing.T) {
	pay, ok := ai.NaivePayment(
		model.AetherPool{Ignus: 1, NonElemental: 1},
		gamecards.Card{Element: "Ignus", CostLevel: "2"},
	)
	if !ok {
		t.Fatal("expected affordable payment")
	}
	if pay.Ignus != 1 || pay.NonElemental != 1 {
		t.Fatalf("payment = %#v; want 1 Ignus + 1 NonElemental", pay)
	}
}

func tinyAIRepository(t *testing.T) *cards.Repository {
	t.Helper()
	path := filepath.Join("..", "..", "..", "data", "cards.json")
	repository, err := cards.LoadFile(path)
	if err != nil {
		t.Fatalf("load cards.json: %v", err)
	}
	return repository
}

func tinyAIDeck(t *testing.T, repository *cards.Repository) decks.Deck {
	t.Helper()
	deck, err := decks.NewDeck("AI Test")
	if err != nil {
		t.Fatal(err)
	}
	remaining := 50
	for _, card := range repository.All() {
		if remaining == 0 {
			break
		}
		if card.ID == "" {
			continue
		}
		qty := 4
		if qty > remaining {
			qty = remaining
		}
		deck.MainDeck = append(deck.MainDeck, decks.DeckEntry{CardID: card.ID, Quantity: qty})
		remaining -= qty
	}
	if remaining != 0 {
		t.Fatalf("could not fill deck (%d short)", remaining)
	}
	if err := deck.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.CanonicalizeCardIDs(repository); err != nil {
		t.Fatal(err)
	}
	return *deck
}
