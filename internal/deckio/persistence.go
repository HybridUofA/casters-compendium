package deckio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// WriteDeck serializes an editable deck as indented JSON.
func WriteDeck(writer io.Writer, deck *decks.Deck) error {
	deckJSON, err := json.MarshalIndent(deck, "", " ")
	if err != nil {
		return fmt.Errorf("encode deck: %w", err)
	}
	if _, err := writer.Write(deckJSON); err != nil {
		return fmt.Errorf("write deck: %w", err)
	}
	return nil
}

// ReadDeck decodes an editable JSON deck and enforces its schema version.
func ReadDeck(reader io.Reader) (*decks.Deck, error) {
	var deck decks.Deck
	if err := json.NewDecoder(reader).Decode(&deck); err != nil {
		return nil, fmt.Errorf("decode deck: %w", err)
	}
	if deck.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported deck schema version: %d", deck.SchemaVersion)
	}
	return &deck, nil
}

// SaveFile writes an editable JSON deck to path.
func SaveFile(path string, deck *decks.Deck) error {
	deckJson, err := json.MarshalIndent(deck, "", " ")
	if err != nil {
		return fmt.Errorf("encode deck: %w", err)
	}

	err = os.WriteFile(path, deckJson, 0644)
	if err != nil {
		return fmt.Errorf("write deck file: %w", err)
	}

	return nil
}

// LoadFile loads an editable JSON deck from path and validates its schema version.
func LoadFile(path string) (*decks.Deck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read deck file: %w", err)
	}

	var deck decks.Deck

	err = json.Unmarshal(data, &deck)
	if err != nil {
		return nil, fmt.Errorf("decode deck file: %w", err)
	}

	if deck.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported deck schema version: %d", deck.SchemaVersion)
	}

	return &deck, nil
}

// ExportDeckList creates a text decklist file and guarantees close errors are reported.
func ExportDeckList(
	path string,
	deck *decks.Deck,
	repository *cards.Repository,
) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create decklist file: %w", err)
	}

	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close decklist file: %w", closeErr)
		}
	}()
	if err := WriteDeckList(file, deck, repository); err != nil {
		return fmt.Errorf("write decklist file: %w", err)
	}

	return nil
}
