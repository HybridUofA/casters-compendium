package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
)

// main caches every missing card image in the bundled development database.
func main() {
	repository, err := cards.LoadFile(
		"data/cards.json",
	)
	if err != nil {
		log.Fatal(err)
	}

	cardList := repository.All()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	ctx := context.Background()

	var downloaded int
	var skipped int
	var failed int

	for index, card := range cardList {
		path, wasDownloaded, downloadErr :=
			cardimages.Download(
				ctx,
				client,
				cardimages.DefaultDirectory,
				card,
			)

		progress := fmt.Sprintf(
			"[%d/%d]",
			index+1,
			len(cardList),
		)

		if downloadErr != nil {
			failed++

			fmt.Printf(
				"%s FAILED: %s: %v\n",
				progress,
				card.Name,
				downloadErr,
			)

			continue
		}

		if !wasDownloaded {
			skipped++

			fmt.Printf(
				"%s SKIPPED: %s\n",
				progress,
				card.Name,
			)

			continue
		}

		downloaded++

		fmt.Printf(
			"%s DOWNLOADED: %s -> %s\n",
			progress,
			card.Name,
			path,
		)

		// Avoid sending hundreds of requests simultaneously
		// or hammering the source website.
		time.Sleep(150 * time.Millisecond)
	}

	fmt.Println()
	fmt.Printf("Downloaded: %d\n", downloaded)
	fmt.Printf("Already cached: %d\n", skipped)
	fmt.Printf("Failed: %d\n", failed)
}
