package deckbuilder

import (
	"context"
	"fmt"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	deckexport "github.com/HybridUofA/casters-compendium/internal/deckbuilder/export"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

const hostedCatalogPointerURL = "https://tts.casterscompendium.com/catalog/current.json"

var hostedCatalogClientFactory = newHostedCatalogClient

func newHostedCatalogClient() (distribution.Client, error) {
	httpClient, err := speedrobo.NewClient()
	if err != nil {
		return distribution.Client{}, err
	}
	return distribution.Client{
		HTTP:       httpClient,
		PointerURL: hostedCatalogPointerURL,
		UserAgent:  cardDatabaseUserAgent,
	}, nil
}

// installPreferredTTSDeck keeps the shipped local exporter as a resilience
// fallback. Hosted export is attempted first because only public HTTPS assets
// can be loaded automatically by other players in a multiplayer room.
func installPreferredTTSDeck(
	ctx context.Context,
	root string,
	deck *decks.Deck,
	repository *cards.Repository,
) (paths deckexport.TTSInstallPaths, hosted bool, fallbackReason error, err error) {
	client, clientErr := hostedCatalogClientFactory()
	if clientErr == nil {
		_, release, releaseErr := client.FetchCurrent(ctx)
		if releaseErr == nil {
			manifest, manifestErr := client.FetchTTSManifest(ctx, release)
			if manifestErr == nil {
				paths, hostedErr := deckexport.InstallHostedTTSDeck(
					root,
					deck,
					manifest,
					repository,
				)
				if hostedErr == nil {
					return paths, true, nil, nil
				}
				fallbackReason = hostedErr
			} else {
				fallbackReason = manifestErr
			}
		} else {
			fallbackReason = releaseErr
		}
	} else {
		fallbackReason = clientErr
	}

	paths, localErr := deckexport.InstallTTSDeck(
		root,
		deck,
		repository,
		cardimages.DefaultDirectory,
		defaultTTSCardBack(),
	)
	if localErr != nil {
		return deckexport.TTSInstallPaths{}, false, fallbackReason, fmt.Errorf(
			"hosted export unavailable (%v); local fallback failed: %w",
			fallbackReason,
			localErr,
		)
	}
	return paths, false, fallbackReason, nil
}
