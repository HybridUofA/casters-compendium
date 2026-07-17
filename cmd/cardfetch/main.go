package main

import (
	"fmt"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"log"
)

func main() {

	repository, err := cards.LoadFile("data/cards.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d cards\n", len(repository.All()))

	card, found := repository.FindByID("104")
	if found {
		fmt.Printf("Found card: %s\n", card.Name)
	}

	matches := repository.SearchByName("passion wing")
	if len(matches) == 0 {
		fmt.Printf("No cards found with the name Passion Wing")
	}

	fmt.Printf("Found %d card(s) named Passion Wing:\n", len(matches))

	for _, match := range matches {
		fmt.Printf(
			"- %s | %s | %s | %s\n",
			match.Name,
			match.CardNumber,
			match.Element,
			match.Expansion,
		)
	}
}
