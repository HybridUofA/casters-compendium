package deckbuilder

import (
	"fmt"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
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
	seed := simulatorPrototypeSeed
	state, err := engine.BeginSetup(engine.SetupInput{
		Players: [2]engine.PlayerSetup{
			{ID: "player-one", Deck: prototypeDeck},
			{ID: "player-two", Deck: prototypeDeck},
		},
		Random:  engine.NewSeededRandom(seed),
		Catalog: repository,
	})
	if err != nil {
		return [2]*session.PlayerSession{}, fmt.Errorf("create simulator prototype match: %w", err)
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
