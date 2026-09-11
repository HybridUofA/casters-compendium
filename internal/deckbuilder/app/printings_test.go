package deckbuilder

import (
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

func TestPrintingOptionLabelIncludesPhysicalPrintingMetadata(t *testing.T) {
	card := cards.Card{
		ID:         "1101",
		CardNumber: "DD01-001",
		Expansion:  "EX01: The Bell Tolls",
		ExtraFields: map[string]string{
			"Rarity": "Rare",
		},
	}

	want := "DD01-001 · EX01: The Bell Tolls · Rare"
	if got := printingOptionLabel(card); got != want {
		t.Fatalf("printingOptionLabel() = %q, want %q", got, want)
	}
}

func TestPrintingOptionLabelFallsBackToID(t *testing.T) {
	if got := printingOptionLabel(cards.Card{ID: "42"}); got != "Printing 42" {
		t.Fatalf("printingOptionLabel() = %q, want Printing 42", got)
	}
}
