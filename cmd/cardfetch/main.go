package main

import (
	"fmt"
	"log"
	"github.com/HybridUofA/caster-deckbuilder/internal/speedrobo"
	"github.com/HybridUofA/caster-deckbuilder/internal/updates"
)

func main() {

	client, err := speedrobo.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	config, err := speedrobo.FetchPageConfig(client)
	if err != nil {
		log.Fatal(err)
	}

	response, err := speedrobo.FetchPage(client, config.AjaxURL, config.Nonce, 1)
	if err != nil {
		log.Fatal(err)
	}

	cards, err := updates.FetchAllCards(client, config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Downloaded %d cards\n", len(cards))

	fmt.Printf("Success: %t\n", response.Success)
	fmt.Printf("Total cards: %d\n", response.Data.Total)
	fmt.Printf("Current page: %d\n", response.Data.Page)
	fmt.Printf("Total pages: %d\n", response.Data.Pages)
	fmt.Printf("Cards received: %d\n", len(response.Data.Cards))

	if len(response.Data.Cards) > 0 {
		fmt.Printf("First card: %s\n", response.Data.Cards[0].CardKey,)
	}
}
