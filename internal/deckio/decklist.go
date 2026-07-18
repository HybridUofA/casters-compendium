// Package deckio reads and writes editable and human-readable deck files.
package deckio

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// ReadDeckList parses the text format produced by WriteDeckList.
func ReadDeckList(reader io.Reader, repository *cards.Repository) (*decks.Deck, error) {
	if reader == nil {
		return nil, fmt.Errorf("decklist reader cannot be nil")
	}
	if repository == nil {
		return nil, fmt.Errorf("card repository cannot be nil")
	}

	scanner := bufio.NewScanner(reader)
	lineNumber := 0
	deckName := ""
	zone := decks.Zone("")
	declaredTotals := make(map[decks.Zone]int)
	seenZones := make(map[decks.Zone]bool)
	deck := &decks.Deck{SchemaVersion: 1}

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Deck Name:") {
			if deckName != "" {
				return nil, decklistLineError(lineNumber, "duplicate deck name")
			}
			deckName = strings.TrimSpace(strings.TrimPrefix(line, "Deck Name:"))
			if deckName == "" {
				return nil, decklistLineError(lineNumber, "deck name cannot be empty")
			}
			continue
		}

		if total, found, err := parseDecklistHeader(line, "Main Deck"); found {
			if err != nil {
				return nil, decklistLineError(lineNumber, err.Error())
			}
			zone = decks.MainZone
			declaredTotals[zone] = total
			seenZones[zone] = true
			continue
		}
		if total, found, err := parseDecklistHeader(line, "Side Deck"); found {
			if err != nil {
				return nil, decklistLineError(lineNumber, err.Error())
			}
			zone = decks.SideZone
			declaredTotals[zone] = total
			seenZones[zone] = true
			continue
		}

		if zone == "" {
			return nil, decklistLineError(lineNumber, "card appears before a deck section")
		}
		quantity, name, expansion, err := parseDecklistCard(line)
		if err != nil {
			return nil, decklistLineError(lineNumber, err.Error())
		}

		card, err := resolveDecklistCard(repository, name, expansion)
		if err != nil {
			return nil, decklistLineError(lineNumber, err.Error())
		}
		if err := deck.AddCard(zone, card.ID, quantity); err != nil {
			return nil, decklistLineError(lineNumber, err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read decklist: %w", err)
	}

	if deckName == "" {
		return nil, fmt.Errorf("decklist has no deck name")
	}
	if !seenZones[decks.MainZone] || !seenZones[decks.SideZone] {
		return nil, fmt.Errorf("decklist must contain main and side deck sections")
	}
	deck.Name = deckName
	if deck.MainTotal() != declaredTotals[decks.MainZone] {
		return nil, fmt.Errorf(
			"main deck declares %d cards but contains %d",
			declaredTotals[decks.MainZone],
			deck.MainTotal(),
		)
	}
	if deck.SideTotal() != declaredTotals[decks.SideZone] {
		return nil, fmt.Errorf(
			"side deck declares %d cards but contains %d",
			declaredTotals[decks.SideZone],
			deck.SideTotal(),
		)
	}
	deck.EnsureOrder()
	return deck, nil
}

// parseDecklistHeader recognizes a named zone header and returns its declared card total.
func parseDecklistHeader(line string, name string) (int, bool, error) {
	prefix := name + " ("
	if !strings.HasPrefix(line, prefix) {
		return 0, false, nil
	}
	if !strings.HasSuffix(line, ")") {
		return 0, true, fmt.Errorf("invalid %s header", strings.ToLower(name))
	}
	totalText := strings.TrimSuffix(strings.TrimPrefix(line, prefix), ")")
	total, err := strconv.Atoi(totalText)
	if err != nil || total < 0 {
		return 0, true, fmt.Errorf("invalid %s total %q", strings.ToLower(name), totalText)
	}
	return total, true, nil
}

// parseDecklistCard parses quantity, card name, and expansion from one decklist line.
func parseDecklistCard(line string) (int, string, string, error) {
	x := strings.Index(line, "x ")
	if x <= 0 {
		return 0, "", "", fmt.Errorf("expected '<quantity>x <name> [<expansion>]'")
	}
	quantity, err := strconv.Atoi(line[:x])
	if err != nil || quantity <= 0 {
		return 0, "", "", fmt.Errorf("invalid card quantity %q", line[:x])
	}

	details := line[x+2:]
	expansionStart := strings.LastIndex(details, " [")
	if expansionStart <= 0 || !strings.HasSuffix(details, "]") {
		return 0, "", "", fmt.Errorf("expected card expansion in brackets")
	}
	name := strings.TrimSpace(details[:expansionStart])
	expansion := strings.TrimSpace(details[expansionStart+2 : len(details)-1])
	if name == "" || expansion == "" {
		return 0, "", "", fmt.Errorf("card name and expansion cannot be empty")
	}
	return quantity, name, expansion, nil
}

// resolveDecklistCard selects the exact named printing from the requested expansion.
func resolveDecklistCard(
	repository *cards.Repository,
	name string,
	expansion string,
) (cards.Card, error) {
	for _, card := range repository.FindByName(name) {
		if strings.EqualFold(strings.TrimSpace(card.Expansion), expansion) {
			return card, nil
		}
	}
	return cards.Card{}, fmt.Errorf("card %q from expansion %q was not found", name, expansion)
}

// decklistLineError annotates a parsing error with its one-based source line.
func decklistLineError(line int, message string) error {
	return fmt.Errorf("decklist line %d: %s", line, message)
}
