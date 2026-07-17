package decks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
)

func SaveFile(path string, deck *Deck) error {
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

func LoadFile(path string) (*Deck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read deck file: %w", err)
	}

	var deck Deck

	err = json.Unmarshal(data, &deck)
	if err != nil {
		return nil, fmt.Errorf("decode deck file: %w", err)
	}

	if deck.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported deck schema version: %d", deck.SchemaVersion)
	}

	return &deck, nil
}

func (deck *Deck) Validate() error {
	if strings.TrimSpace(deck.Name) == "" {
		return fmt.Errorf("deck name cannot be empty")
	}

	if deck.SchemaVersion != 1 {
		return fmt.Errorf(
			"unsupported schema version: %d",
			deck.SchemaVersion,
		)
	}
	return nil
}

func (deck *Deck) ValidateCards(repository *cards.Repository) error {
	for _, entry := range deck.MainDeck {
		_, found := repository.FindByID(entry.CardID)
		if !found {
			return fmt.Errorf(
				"main deck contains unknown card ID %q",
				entry.CardID,
			)
		}
	}
	return nil
}

func ExportDeckList(
	path string,
	deck *Deck,
	repository *cards.Repository,
) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create decklist file: %w", err)
	}

	defer file.Close()
	WriteDeckList(file, deck, repository)

	return nil
}
