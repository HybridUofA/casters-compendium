package main

import (
	"strings"
	"testing"
)

// TestHowToUseGuideDocumentsCardRemoval protects the non-obvious deck interaction.
func TestHowToUseGuideDocumentsCardRemoval(t *testing.T) {
	if !strings.Contains(
		howToUseMarkdown,
		"Right-click a card already in the Main Deck or Side Deck to remove one copy.",
	) {
		t.Fatal("How to Use guide does not explain how to remove a card")
	}
}
