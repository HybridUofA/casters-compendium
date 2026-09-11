package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	cardupdate "github.com/HybridUofA/casters-compendium/internal/carddata/update"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

// main converts raw Speedrobo card details into the normalized shared card schema.
func main() {
	previousData, previousErr := os.ReadFile("data/cards.json")
	var previous []cards.Card
	if previousErr == nil {
		if err := json.Unmarshal(previousData, &previous); err != nil {
			log.Fatalf("decode previous normalized card database: %v", err)
		}
	} else if !os.IsNotExist(previousErr) {
		log.Fatalf("read previous normalized card database: %v", previousErr)
	}

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
	normalized, migrations := cardupdate.CarryForwardLegacyIDs(previous, normalized)
	if _, err := catalog.NewRepository(normalized); err != nil {
		log.Fatalf("validate normalized repository: %v", err)
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
	for _, migration := range migrations {
		fmt.Printf("Preserved legacy card ID %s as %s\n", migration.LegacyID, migration.CanonicalID)
	}
}
