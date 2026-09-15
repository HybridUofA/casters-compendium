package decks

import (
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

type copyLimitCatalog map[string]gamecards.Card

func (catalog copyLimitCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, found := catalog[id]
	return card, found
}

// TestAddCardCheckedSeparatesCasterLevels verifies Level 1 and Level 2 copies
// have independent four-copy limits while alternate printings of one level do
// not bypass that limit.
func TestAddCardCheckedSeparatesCasterLevels(t *testing.T) {
	levelOne := gamecards.Card{ID: "arthur-1", Name: "Arthur", Type: "Caster", CostLevel: "1"}
	levelOneAlt := gamecards.Card{ID: "arthur-1-alt", Name: "Arthur", Type: "Caster", CostLevel: "1"}
	levelTwo := gamecards.Card{ID: "arthur-2", Name: "Arthur", Type: "Caster", CostLevel: "2"}
	catalog := copyLimitCatalog{
		levelOne.ID:    levelOne,
		levelOneAlt.ID: levelOneAlt,
		levelTwo.ID:    levelTwo,
	}
	deck, err := NewDeck("Caster Levels")
	if err != nil {
		t.Fatal(err)
	}

	added, err := deck.AddCardChecked(MainZone, levelOne, 4, catalog)
	if err != nil || !added {
		t.Fatalf("add four Level 1 copies = %t, %v", added, err)
	}
	added, err = deck.AddCardChecked(MainZone, levelOneAlt, 1, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("alternate Level 1 printing bypassed the four-copy limit")
	}

	added, err = deck.AddCardChecked(MainZone, levelTwo, 4, catalog)
	if err != nil || !added {
		t.Fatalf("add four Level 2 copies = %t, %v", added, err)
	}
	if got := deck.CopiesOfCard(levelOne, catalog); got != 4 {
		t.Fatalf("Level 1 copies = %d, want 4", got)
	}
	if got := deck.CopiesOfCard(levelTwo, catalog); got != 4 {
		t.Fatalf("Level 2 copies = %d, want 4", got)
	}
}
