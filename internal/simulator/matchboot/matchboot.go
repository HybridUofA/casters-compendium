// Package matchboot builds authoritative LocalMatch instances for networked play.
package matchboot

import (
	"fmt"
	"math/rand/v2"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/nethost"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

// LoadCatalog loads a normalized cards.json database.
func LoadCatalog(path string) (*cards.Repository, error) {
	repository, err := cards.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load card catalog: %w", err)
	}
	return repository, nil
}

// PrototypeFactory returns a MatchFactory that starts each room with two copies
// of the 50-card simulator prototype deck and a fresh random seed.
func PrototypeFactory(repository *cards.Repository) nethost.MatchFactory {
	return func(room *nethost.Room) (*session.LocalMatch, error) {
		_ = room
		return NewPrototypeMatch(repository, NewSeed())
	}
}

// SubmittedDecksFactory starts each room using the decks players submitted on
// create/join. Both seats must include a validated main deck.
func SubmittedDecksFactory(repository *cards.Repository) nethost.MatchFactory {
	return func(room *nethost.Room) (*session.LocalMatch, error) {
		if repository == nil {
			return nil, fmt.Errorf("card repository cannot be nil")
		}
		if room == nil {
			return nil, fmt.Errorf("room cannot be nil")
		}
		playerOneDeck, ok := room.Decks["player-one"]
		if !ok {
			return nil, fmt.Errorf("player-one deck is missing")
		}
		playerTwoDeck, ok := room.Decks["player-two"]
		if !ok {
			return nil, fmt.Errorf("player-two deck is missing")
		}
		if err := PrepareSubmittedDeck(&playerOneDeck, repository); err != nil {
			return nil, fmt.Errorf("player-one deck: %w", err)
		}
		if err := PrepareSubmittedDeck(&playerTwoDeck, repository); err != nil {
			return nil, fmt.Errorf("player-two deck: %w", err)
		}
		return NewMatch(repository, [2]decks.Deck{playerOneDeck, playerTwoDeck}, NewSeed())
	}
}

// PrepareSubmittedDeck validates and canonicalizes a client-submitted deck.
func PrepareSubmittedDeck(deck *decks.Deck, repository *cards.Repository) error {
	if deck == nil {
		return fmt.Errorf("deck is required")
	}
	if repository == nil {
		return fmt.Errorf("card repository cannot be nil")
	}
	if err := deck.Validate(); err != nil {
		return err
	}
	if _, err := deck.CanonicalizeCardIDs(repository); err != nil {
		return err
	}
	if err := deck.ValidateCards(repository); err != nil {
		return err
	}
	return nil
}

// NewSeed returns a non-deterministic match seed for live games.
func NewSeed() engine.MatchSeed {
	return engine.MatchSeed{First: rand.Uint64(), Second: rand.Uint64()}
}

// NewPrototypeMatch builds one LocalMatch using two prototype decks.
func NewPrototypeMatch(repository *cards.Repository, seed engine.MatchSeed) (*session.LocalMatch, error) {
	if repository == nil {
		return nil, fmt.Errorf("card repository cannot be nil")
	}
	deck, err := BuildPrototypeDeck(repository)
	if err != nil {
		return nil, err
	}
	return NewMatch(repository, [2]decks.Deck{deck, deck}, seed)
}

// NewMatch runs engine setup and wraps the result in a LocalMatch.
func NewMatch(
	repository *cards.Repository,
	playerDecks [2]decks.Deck,
	seed engine.MatchSeed,
) (*session.LocalMatch, error) {
	if repository == nil {
		return nil, fmt.Errorf("card repository cannot be nil")
	}
	state, err := engine.BeginSetup(engine.SetupInput{
		Players: [2]engine.PlayerSetup{
			{ID: "player-one", Deck: playerDecks[0]},
			{ID: "player-two", Deck: playerDecks[1]},
		},
		Random:  engine.NewSeededRandom(seed),
		Catalog: repository,
	})
	if err != nil {
		return nil, fmt.Errorf("begin setup: %w", err)
	}
	match, err := session.NewLocalMatch(state, seed, repository)
	if err != nil {
		return nil, fmt.Errorf("create local match: %w", err)
	}
	return match, nil
}

// BuildPrototypeDeck mirrors the hotseat simulator prototype list.
func BuildPrototypeDeck(repository *cards.Repository) (decks.Deck, error) {
	const prototypeDeckSize = 50

	deck, err := decks.NewDeck("Simulator Prototype")
	if err != nil {
		return decks.Deck{}, err
	}

	remaining := prototypeDeckSize
	selected := make(map[string]struct{})

	for _, wantedLevel := range []string{"1", "2"} {
		for _, card := range repository.All() {
			if !strings.EqualFold(strings.TrimSpace(card.Name), "Arthur") ||
				!strings.EqualFold(strings.TrimSpace(card.Type), "Caster") ||
				strings.TrimSpace(card.CostLevel) != wantedLevel {
				continue
			}
			quantity := min(decks.MaxCopiesPerCard, remaining)
			deck.MainDeck = append(deck.MainDeck, decks.DeckEntry{
				CardID:   card.ID,
				Quantity: quantity,
			})
			selected[card.ID] = struct{}{}
			remaining -= quantity
			break
		}
	}

	for _, card := range repository.All() {
		if remaining == 0 {
			break
		}
		if strings.TrimSpace(card.ID) == "" ||
			strings.EqualFold(strings.TrimSpace(card.Name), "Caster Token") {
			continue
		}
		if _, alreadySelected := selected[card.ID]; alreadySelected {
			continue
		}
		quantity := min(decks.MaxCopiesPerCard, remaining)
		deck.MainDeck = append(deck.MainDeck, decks.DeckEntry{
			CardID:   card.ID,
			Quantity: quantity,
		})
		remaining -= quantity
	}
	if remaining != 0 {
		return decks.Deck{}, fmt.Errorf(
			"card repository does not contain enough definitions for a %d-card prototype deck",
			prototypeDeckSize,
		)
	}
	deck.EnsureOrder()
	return *deck, nil
}
