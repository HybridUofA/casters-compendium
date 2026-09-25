package deckbuilder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSimulatorDeckChoicesIncludesPersonalJSONAndTextDecks(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"My Fire Deck.json", "Tournament List.txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("deck"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	choices, err := simulatorDeckChoices(directory)
	if err != nil {
		t.Fatal(err)
	}
	foundJSON := false
	foundText := false
	for _, choice := range choices {
		switch filepath.Base(choice.path) {
		case "My Fire Deck.json":
			foundJSON = true
		case "Tournament List.txt":
			foundText = true
		}
	}
	if !foundJSON || !foundText {
		t.Fatalf("personal simulator choices found JSON/text = %t/%t", foundJSON, foundText)
	}
}
