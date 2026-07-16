package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/speedrobo"
)

func main() {
	rawData, err := os.ReadFile("data/cards.raw.json")
	if err != nil {
		log.Fatal("read raw card database: %v", err)
	}

	var details []speedrobo.CardDetail

	if err := json.Unmarshal(rawData, &details); err != nil {
		log.Fatalf("decode raw card database: %v", err)
	}

	normalized := make([]cards.Card, 0, len(details))

	for _, detail := range details {
		card, err := cards.FromSpeedrobo(detail)
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