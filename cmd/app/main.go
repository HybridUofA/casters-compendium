package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
)

// main provides the legacy command entry point; interactive behavior is exposed through InitCLI.
func main() {
}

// InitCLI runs the line-oriented deck editor until input closes or the user quits.
func InitCLI(repository *cards.Repository, deck *decks.Deck) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := strings.ToLower(parts[0])

		switch command {
		case "search":
			handleSearch(parts, repository)

		case "add":
			handleAdd(parts, deck, repository)

		case "remove":
			handleRemove(parts, deck, repository)

		case "list":
			handleList(deck, repository)

		case "quit", "exit", "q":
			return

		default:
			fmt.Printf("unknown command: %s\n", command)
		}
	}
}

// handleSearch prints cards whose names contain the requested search terms.
func handleSearch(
	args []string,
	repository *cards.Repository,
) {
	if len(args) < 2 {
		fmt.Println("usage: search <card name>")
		return
	}
	query := strings.Join(args[1:], " ")
	matches := repository.SearchByName(query)
	if len(matches) == 0 {
		fmt.Printf("No cards found matching %q.\n", query)
		return
	}
	for _, match := range matches {
		fmt.Printf(
			"- %s | %s | %s | ID: %s | %s\n",
			match.Name,
			match.Type,
			match.Element,
			match.ID,
			match.Expansion,
		)
	}
}

// handleAdd validates an add command and places copies in the requested deck zone.
func handleAdd(
	args []string,
	deck *decks.Deck,
	repository *cards.Repository,
) {
	if len(args) != 4 {
		fmt.Println("usage: add <main|side> <card-id> <quantity>")
		return
	}

	section := strings.ToLower(args[1])
	cardID := args[2]
	quantityText := args[3]

	quantity, err := strconv.Atoi(quantityText)
	if err != nil || quantity <= 0 {
		fmt.Println("quantity must be a positive integer")
		return
	}

	card, found := repository.FindByID(cardID)
	if !found {
		fmt.Printf("card ID %q was not found\n", cardID)
		return
	}

	zone := decks.Zone(strings.ToLower(args[1]))
	err = deck.AddCard(zone, cardID, quantity)
	if err != nil {
		message := "could not add card: %v\n"
		fmt.Println(message, err)
		return
	}

	fmt.Printf(
		"Added %dx %s to the %s deck.\n",
		quantity,
		card.Name,
		section,
	)
}

// handleRemove validates a remove command and removes copies from the requested deck zone.
func handleRemove(
	args []string,
	deck *decks.Deck,
	repository *cards.Repository,
) {
	if len(args) != 4 {
		fmt.Println("usage: remove <main|side> <card-id> <quantity>")
		return
	}

	section := strings.ToLower(args[1])
	cardID := args[2]
	quantityText := args[3]

	quantity, err := strconv.Atoi(quantityText)
	if err != nil || quantity <= 0 {
		fmt.Println("quantity must be a positive integer")
		return
	}

	card, found := repository.FindByID(cardID)
	if !found {
		fmt.Printf("card ID %q was not found\n", cardID)
		return
	}

	zone := decks.Zone(strings.ToLower(args[1]))
	err = deck.RemoveCard(zone, cardID, quantity)
	if err != nil {
		message := "could not remove card: %v\n"
		fmt.Println(message, err)
		return
	}

	fmt.Printf(
		"removed %dx %s from the %s deck.\n",
		quantity,
		card.Name,
		section,
	)
}

// handleList writes the current deck in the human-readable decklist format.
func handleList(
	deck *decks.Deck,
	repository *cards.Repository,
) {
	decks.WriteDeckList(os.Stdout, deck, repository)
}
