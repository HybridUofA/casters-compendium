package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	cardupdate "github.com/HybridUofA/casters-compendium/internal/carddata/update"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

// main converts raw Speedrobo card details into the normalized shared card schema.
func main() {
	rawData, err := os.ReadFile("data/cards.raw.json")
	if err != nil {
		message := "read raw card database: %v"
		log.Fatal(message, err)
	}

	var details []speedrobo.CardDetail

	if err := json.Unmarshal(rawData, &details); err != nil {
		log.Fatalf("decode raw card database: %v", err)
	}

	normalized := make([]cards.Card, 0, len(details))

	for _, detail := range details {
		card, err := cardupdate.FromSpeedrobo(detail)
		if err != nil {
			log.Fatalf(
				"normalize card %q (%s): %v",
				detail.CardKey,
				detail.ID,
				err,
			)
		}

		normalized = append(normalized, card)
	}

	output, err := json.MarshalIndent(normalized, "", " ")
	if err != nil {
		log.Fatalf("encode normalized card database: %v", err)
	}

	output = append(output, '\n')

	if err := os.WriteFile("data/cards.json", output, 0644); err != nil {
		log.Fatalf("write normalized card database: %v", err)
	}

	fmt.Printf("Normalized %d cards\n", len(normalized))
}
