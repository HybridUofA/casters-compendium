package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
)

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
		fmt.Println("could not add card: %v\n", err)
		return
	}

	fmt.Printf(
		"Added %dx %s to the %s deck.\n",
		quantity,
		card.Name,
		section,
	)
}

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
		fmt.Println("could not remove card: %v\n", err)
		return
	}

	fmt.Printf(
		"removed %dx %s from the %s deck.\n",
		quantity,
		card.Name,
		section,
	)
}

func handleList(
	deck *decks.Deck,
	repository *cards.Repository,
) {
	decks.WriteDeckList(os.Stdout, deck, repository)
}
