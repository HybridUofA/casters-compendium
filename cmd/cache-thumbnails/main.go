package main

import (
	"fmt"
	"log"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
)

// main creates thumbnails for every card image that does not already have one.
func main() {
	repository, err := cards.LoadFile("data/cards.json")
	if err != nil {
		log.Fatal(err)
	}

	cardList := repository.All()

	var created int
	var skipped int
	var failed int

	for index, card := range cardList {
		progress := fmt.Sprintf(
			"[%d/%d]",
			index+1,
			len(cardList),
		)

		if _, found := cardimages.FindThumbnail(card.ID); found {
			skipped++
			fmt.Printf(
				"%s SKIPPED: %s\n",
				progress,
				card.Name,
			)
			continue
		}

		path, err := cardimages.CreateThumbnail(card.ID)
		if err != nil {
			failed++
			fmt.Printf(
				"%s FAILED: %s: %v\n",
				progress,
				card.Name,
				err,
			)
			continue
		}

		created++
		fmt.Printf(
			"%s CREATED: %s -> %s\n",
			progress,
			card.Name,
			path,
		)
	}

	fmt.Println()
	fmt.Printf("Created: %d\n", created)
	fmt.Printf("Skipped: %d\n", skipped)
	fmt.Printf("Failed: %d\n", failed)
}
