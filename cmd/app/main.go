package main

import (
	"fmt"
	"log"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/cli"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
)

func main() {

	repository, err := cards.LoadFile("data/cards.json")
	if err != nil {
		log.Fatal(err)
	}

	deck, err := decks.NewDeck("New Deck")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d cards\n", len(repository.All()))

	cli.InitCLI(repository, deck)
}
