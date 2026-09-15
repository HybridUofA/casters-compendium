package deckio

import (
	"bytes"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

func TestDeckPersistenceRetainsCustomTTSCardBack(t *testing.T) {
	want := "https://custom-assets.casterscompendium.com/assets/tts-card-back/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	deck := &decks.Deck{SchemaVersion: 1, Name: "Custom", TTSCardBackURL: want}
	var encoded bytes.Buffer
	if err := WriteDeck(&encoded, deck); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadDeck(&encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.TTSCardBackURL != want {
		t.Fatalf("TTSCardBackURL = %q, want %q", decoded.TTSCardBackURL, want)
	}
}
