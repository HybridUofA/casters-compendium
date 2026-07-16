package main

import (
	"fmt"
	"net/http"
	"log"
	"os"
	"github.com/HybridUofA/caster-deckbuilder/internal/speedrobo"
	"time"
)

func main() {
	nonce := os.Getenv("SPEEDROBO_NONCE")
	if nonce == "" {
		log.Fatal("SPEEDROBO_NONCE is not set")
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	response, err := speedrobo.FetchPage(client, nonce, 1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Success: %t\n", response.Success)
	fmt.Printf("Total cards: %d\n", response.Data.Total)
	fmt.Printf("Current page: %d\n", response.Data.Page)
	fmt.Printf("Total pages: %d\n", response.Data.Pages)
	fmt.Printf("Cards received: %d\n", len(response.Data.Cards))

	if len(response.Data.Cards) > 0 {
		fmt.Printf("First card: %s\n", response.Data.Cards[0].CardKey,)
	}
}
