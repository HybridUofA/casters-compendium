package deckbuilder

import (
	"strings"
	"testing"
)

// TestHowToUseGuideDocumentsCardRemoval protects the non-obvious deck interaction.
func TestHowToUseGuideDocumentsCardRemoval(t *testing.T) {
	for _, instruction := range []string{
		"Right-click a card already in the Main Deck or Side Deck to remove one copy.",
		"Drag a deck card onto the **Card Search** panel to remove one copy.",
	} {
		if !strings.Contains(howToUseMarkdown, instruction) {
			t.Fatalf("How to Use guide is missing removal instruction %q", instruction)
		}
	}
}

func TestHowToUseGuideDocumentsArtworkSelection(t *testing.T) {
	for _, instruction := range []string{
		"Artwork / Printing",
		"Show each artwork separately",
	} {
		if !strings.Contains(howToUseMarkdown, instruction) {
			t.Fatalf("How to Use guide is missing artwork instruction %q", instruction)
		}
	}
}

func TestHowToUseGuideDocumentsHistoricalIDMigration(t *testing.T) {
	if !strings.Contains(howToUseMarkdown, "historical card ID") {
		t.Fatal("How to Use guide is missing historical card ID migration guidance")
	}
}
