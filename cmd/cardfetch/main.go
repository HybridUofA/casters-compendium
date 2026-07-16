package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io"
	"log"
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

	if len(reponse.Data.Cards) > 0 {
		fmt.Printf("First card: %s\n", response.Data.Cards[0].CardKey,)
	}
}
