// Command cardrefresh downloads and normalizes the current Speedrobo card list.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	cardupdate "github.com/HybridUofA/casters-compendium/internal/carddata/update"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

func main() {
	rawPath := flag.String("raw", "data/cards.raw.json", "raw card-detail output path")
	normalizedPath := flag.String("normalized", "data/cards.json", "normalized card output path")
	flag.Parse()

	client, err := speedrobo.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	config, err := speedrobo.FetchPageConfig(client)
	if err != nil {
		log.Fatal(err)
	}
	summaries, err := speedrobo.FetchAllCards(client, config)
	if err != nil {
		log.Fatal(err)
	}
	details, err := speedrobo.FetchAllCardDetails(client, config, summaries)
	if err != nil {
		log.Fatal(err)
	}

	normalized := make([]cards.Card, 0, len(details))
	for _, detail := range details {
		card, normalizeErr := cardupdate.FromSpeedrobo(detail)
		if normalizeErr != nil {
			log.Fatalf("normalize %q (%s): %v", detail.CardKey, detail.ID, normalizeErr)
		}
		normalized = append(normalized, card)
	}
	if _, err := catalog.NewRepository(normalized); err != nil {
		log.Fatalf("validate normalized repository: %v", err)
	}

	rawData, err := marshalJSON(details)
	if err != nil {
		log.Fatalf("encode raw card database: %v", err)
	}
	normalizedData, err := marshalJSON(normalized)
	if err != nil {
		log.Fatalf("encode normalized card database: %v", err)
	}
	if err := writeAtomically(*rawPath, rawData); err != nil {
		log.Fatalf("write raw card database: %v", err)
	}
	if err := writeAtomically(*normalizedPath, normalizedData); err != nil {
		log.Fatalf("write normalized card database: %v", err)
	}

	fmt.Printf("Refreshed %d cards\n", len(normalized))
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", " ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func writeAtomically(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".card-refresh-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
