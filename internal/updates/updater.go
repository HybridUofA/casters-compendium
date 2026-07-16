package updates

import (
	"net/http"
	"github.com/HybridUofA/caster-deckbuilder/internal/speedrobo"
	"fmt"
)

func FetchAllCards(
	client *http.Client,
	config speedrobo.PageConfig,
) ([]speedrobo.CardResponse, error) {
	firstResponse, err := speedrobo.FetchPage(client, config.AjaxURL, config.Nonce, 1)
	if err != nil {
		return nil, fmt.Errorf("error occurred on page 1: %w", err)
	}

	allCards := append(
		[]speedrobo.CardResponse{},
		firstResponse.Data.Cards...,
	)

	for page := 2; page <= firstResponse.Data.Pages; page++ {
		pageResponse, err := speedrobo.FetchPage(
			client,
			config.AjaxURL,
			config.Nonce,
			page,
		)
		if err != nil {
			return nil, fmt.Errorf("error occurred on page %d: %w", page, err)
		}

		if pageResponse.Data.Page != page {
			return nil, fmt.Errorf(
				"requested page %d but received page %d",
				page,
				pageResponse.Data.Page,
			)
		}

		allCards = append(allCards, pageResponse.Data.Cards...,)
	}

	if len(allCards) != firstResponse.Data.Total {
		return nil, fmt.Errorf(
			"card count mismatch: expected %d, received %d",
			firstResponse.Data.Total,
			len(allCards),
		)
	}

	return allCards, nil
}