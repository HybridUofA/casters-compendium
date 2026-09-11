package rules

import (
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestCalculateCasterAetherMapsEveryElement(t *testing.T) {
	tests := []struct {
		name        string
		element     string
		level       string
		wantElement model.Element
		wantAmount  int
	}{
		{name: "Aes", element: "Aes", level: "1", wantElement: model.ElementAes, wantAmount: 1},
		{name: "Aqua normalized", element: " aqua ", level: " 2 ", wantElement: model.ElementAqua, wantAmount: 2},
		{name: "Ignus", element: "Ignus", level: "3", wantElement: model.ElementIgnus, wantAmount: 3},
		{name: "Luna", element: "Luna", level: "4", wantElement: model.ElementLuna, wantAmount: 4},
		{name: "Silva", element: "Silva", level: "2", wantElement: model.ElementSilva, wantAmount: 2},
		{name: "Solis", element: "Solis", level: "1", wantElement: model.ElementSolis, wantAmount: 1},
		{name: "Terra", element: "Terra", level: "3", wantElement: model.ElementTerra, wantAmount: 3},
		{name: "Void", element: "Void", level: "4", wantElement: model.ElementVoid, wantAmount: 4},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			const cardID = "test-caster"
			catalog := definitionCatalogForTest{
				cardID: {
					ID:        cardID,
					Type:      " cAsTeR ",
					Element:   testCase.element,
					CostLevel: testCase.level,
				},
			}

			element, amount, err := CalculateCasterAether(catalog, cardID)
			if err != nil {
				t.Fatalf("CalculateCasterAether() error = %v; want nil", err)
			}
			if element != testCase.wantElement || amount != testCase.wantAmount {
				t.Fatalf("CalculateCasterAether() = %q/%d; want %q/%d", element, amount, testCase.wantElement, testCase.wantAmount)
			}
		})
	}
}

func TestCalculateCasterAetherRejectsInvalidDefinitions(t *testing.T) {
	validCard := gamecards.Card{
		ID:        "test-caster",
		Type:      "Caster",
		Element:   "Aes",
		CostLevel: "1",
	}
	tests := []struct {
		name        string
		catalog     CardCatalog
		cardID      model.CardID
		wantErrPart string
	}{
		{name: "nil catalog", cardID: model.CardID(validCard.ID), wantErrPart: "catalog cannot be nil"},
		{name: "blank card ID", catalog: definitionCatalogForTest{}, cardID: " ", wantErrPart: "cardID cannot be empty"},
		{name: "missing card", catalog: definitionCatalogForTest{}, cardID: "missing", wantErrPart: "error looking up card"},
		{name: "non-Caster type", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.Type = "Servant"
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "not a caster"},
		{name: "nonnumeric Level", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.CostLevel = "X"
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "level invalid"},
		{name: "zero Level", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.CostLevel = "0"
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "must be positive"},
		{name: "negative Level", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.CostLevel = "-2"
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "must be positive"},
		{name: "unknown Element", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.Element = "Cheese"
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "blank or unknown"},
		{name: "blank Element", catalog: definitionCatalogForTest{validCard.ID: func() gamecards.Card {
			card := validCard
			card.Element = " "
			return card
		}()}, cardID: model.CardID(validCard.ID), wantErrPart: "blank or unknown"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			element, amount, err := CalculateCasterAether(testCase.catalog, testCase.cardID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("CalculateCasterAether() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if element != "" || amount != 0 {
				t.Fatalf("CalculateCasterAether() = %q/%d on error; want zero values", element, amount)
			}
		})
	}
}
