package cards

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Repository struct {
	cards  []Card
	byID   map[string]Card
	byName map[string][]Card
}

func LoadFile(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read card database %q: %w", path, err)
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("decode card database %q: %w", path, err)
	}

	repository, err := NewRepository(cards)
	if err != nil {
		return nil, fmt.Errorf(
			"build repository from %q, %w",
			path,
			err,
		)
	}

	return repository, nil
}

func NewRepository(cards []Card) (*Repository, error) {
	if len(cards) == 0 {
		return nil, fmt.Errorf("card list cannot be empty")
	}

	repository := &Repository{
		cards:  make([]Card, 0, len(cards)),
		byID:   make(map[string]Card, len(cards)),
		byName: make(map[string][]Card),
	}

	for index, card := range cards {
		card.ID = strings.TrimSpace(card.ID)
		card.Name = strings.TrimSpace(card.Name)

		if card.ID == "" {
			return nil, fmt.Errorf(
				"card at index %d has no ID",
				index,
			)
		}

		if card.Name == "" {
			return nil, fmt.Errorf(
				"card %q has no name",
				card.ID,
			)
		}

		if _, exists := repository.byID[card.ID]; exists {
			return nil, fmt.Errorf(
				"duplicate card ID %q",
				card.ID,
			)
		}

		repository.cards = append(repository.cards, card)
		repository.byID[card.ID] = card

		nameKey := normalizeName(card.Name)

		repository.byName[nameKey] = append(
			repository.byName[nameKey],
			card,
		)
	}

	return repository, nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (repository *Repository) FindByID(id string) (Card, bool) {
	card, found := repository.byID[strings.TrimSpace(id)]
	return card, found
}

func (repository *Repository) FindByName(name string) []Card {
	matches := repository.byName[normalizeName(name)]

	result := make([]Card, len(matches))
	copy(result, matches)

	return result
}

func (repository *Repository) SearchByName(query string) []Card {
	normalizedQuery := normalizeName(query)

	if normalizedQuery == "" {
		return []Card{}
	}

	var matches []Card

	for _, card := range repository.cards {
		normalizedCardName := normalizeName(card.Name)

		if strings.Contains(normalizedCardName, normalizedQuery) {
			matches = append(matches, card)
		}
	}

	return matches
}

func (repository *Repository) All() []Card {
	result := make([]Card, len(repository.cards))
	copy(result, repository.cards)

	return result
}
