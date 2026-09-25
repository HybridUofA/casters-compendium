package deckbuilder

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/deckio"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

var simulatorPrototypeSeed = engine.MatchSeed{
	First:  1,
	Second: 2,
}

// buildSimulatorPrototypeSessions creates one authoritative setup snapshot and
// attaches a private session for each player.
func buildSimulatorPrototypeSessions(
	repository *cards.Repository,
) ([2]*session.PlayerSession, error) {
	if repository == nil {
		return [2]*session.PlayerSession{}, fmt.Errorf("card repository cannot be nil")
	}

	prototypeDeck, err := buildPrototypeDeck(repository)
	if err != nil {
		return [2]*session.PlayerSession{}, err
	}
	return buildSimulatorSessions(repository, [2]decks.Deck{prototypeDeck, prototypeDeck}, simulatorPrototypeSeed)
}

func buildSimulatorSessions(
	repository *cards.Repository,
	playerDecks [2]decks.Deck,
	seed engine.MatchSeed,
) ([2]*session.PlayerSession, error) {
	if repository == nil {
		return [2]*session.PlayerSession{}, fmt.Errorf("card repository cannot be nil")
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
		return [2]*session.PlayerSession{}, fmt.Errorf("create simulator match: %w", err)
	}

	localMatch, err := session.NewLocalMatch(state, seed, repository)
	if err != nil {
		return [2]*session.PlayerSession{}, fmt.Errorf("create local simulator match: %w", err)
	}
	playerSessions := [2]*session.PlayerSession{}
	for index, player := range state.Players {
		playerSessions[index], err = session.NewPlayerSession(localMatch, player.ID)
		if err != nil {
			return [2]*session.PlayerSession{}, fmt.Errorf(
				"create simulator session for player %d: %w",
				index+1,
				err,
			)
		}
	}
	return playerSessions, nil
}

func newSimulatorMatchSeed() engine.MatchSeed {
	return engine.MatchSeed{First: rand.Uint64(), Second: rand.Uint64()}
}

func loadSimulatorDeck(path string, repository *cards.Repository) (*decks.Deck, error) {
	if repository == nil {
		return nil, fmt.Errorf("card repository cannot be nil")
	}
	var deck *decks.Deck
	var err error
	if strings.EqualFold(filepath.Ext(path), ".txt") {
		reader, openErr := os.Open(path)
		if openErr != nil {
			return nil, fmt.Errorf("open simulator deck: %w", openErr)
		}
		deck, err = deckio.ReadDeckList(reader, repository)
		closeErr := reader.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	} else {
		deck, err = deckio.LoadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("load simulator deck: %w", err)
	}
	if _, err := deck.CanonicalizeCardIDs(repository); err != nil {
		return nil, fmt.Errorf("canonicalize simulator deck: %w", err)
	}
	return deck, nil
}

func buildPrototypeDeck(repository *cards.Repository) (decks.Deck, error) {
	const prototypeDeckSize = 50

	deck, err := decks.NewDeck("Simulator Prototype")
	if err != nil {
		return decks.Deck{}, err
	}

	remaining := prototypeDeckSize
	selected := make(map[string]struct{})

	// Put both printed Arthur levels into the prototype list before filling it.
	// The fixed prototype seed then makes the pair available in Player One's
	// opening hand for visually exercising the Level Up interaction.
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
