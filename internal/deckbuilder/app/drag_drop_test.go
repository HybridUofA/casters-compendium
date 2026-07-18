package deckbuilder

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	deckui "github.com/HybridUofA/casters-compendium/internal/deckbuilder/ui"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// TestApplyCardDropRemovesDeckCard verifies that the search-panel drop target removes one physical copy.
func TestApplyCardDropRemovesDeckCard(t *testing.T) {
	repository, err := catalog.NewRepository([]catalog.Card{
		{ID: "test-card", Name: "Test Card"},
	})
	if err != nil {
		t.Fatal(err)
	}

	deck, err := decks.NewDeck("Drag Removal Test")
	if err != nil {
		t.Fatal(err)
	}

	card, found := repository.FindByID("test-card")
	if !found {
		t.Fatal("test card was not added to the repository")
	}

	added, err := deck.AddCardChecked(
		decks.MainZone,
		card,
		2,
		repository,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("test card was not added to the deck")
	}

	err = applyCardDrop(
		deck,
		repository,
		deckui.CardDragSource{
			Kind:  deckui.DragFromDeck,
			Card:  card,
			Zone:  decks.MainZone,
			Index: 0,
		},
		&deckui.CardDropTarget{Remove: true},
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := deck.MainTotal(); got != 1 {
		t.Fatalf("main-deck total = %d, want 1", got)
	}
	if got := len(deck.MainOrder); got != 1 {
		t.Fatalf("main-deck order length = %d, want 1", got)
	}
}

// TestApplyCardDropIgnoresSearchCardRemoval verifies that search results cannot remove deck cards.
func TestApplyCardDropIgnoresSearchCardRemoval(t *testing.T) {
	repository, err := catalog.NewRepository([]catalog.Card{
		{ID: "test-card", Name: "Test Card"},
	})
	if err != nil {
		t.Fatal(err)
	}

	deck, err := decks.NewDeck("Search Source Test")
	if err != nil {
		t.Fatal(err)
	}

	card, found := repository.FindByID("test-card")
	if !found {
		t.Fatal("test card was not added to the repository")
	}

	if err := applyCardDrop(
		deck,
		repository,
		deckui.CardDragSource{
			Kind: deckui.DragFromSearch,
			Card: card,
		},
		&deckui.CardDropTarget{Remove: true},
	); err != nil {
		t.Fatal(err)
	}

	if got := deck.MainTotal(); got != 0 {
		t.Fatalf("main-deck total = %d, want 0", got)
	}
}
