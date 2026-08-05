package rules

import (
	"reflect"
	"strings"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

type definitionCatalogForTest map[string]gamecards.Card

func (catalog definitionCatalogForTest) FindByID(id string) (gamecards.Card, bool) {
	card, found := catalog[id]
	return card, found
}

func TestResolveCardDefinitionReturnsCatalogCard(t *testing.T) {
	want := gamecards.Card{
		ID:        "caster-1",
		Name:      "Test Caster",
		Type:      "Caster",
		Element:   "Aqua",
		CostLevel: "2",
	}

	got, err := ResolveCardDefinition(
		definitionCatalogForTest{want.ID: want},
		model.CardID(want.ID),
	)
	if err != nil {
		t.Fatalf("ResolveCardDefinition() error = %v; want nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveCardDefinition() = %#v; want %#v", got, want)
	}
}

func TestResolveCardDefinitionRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		catalog     CardCatalog
		cardID      model.CardID
		wantErrPart string
	}{
		{name: "nil catalog", cardID: "caster-1", wantErrPart: "catalog cannot be nil"},
		{name: "blank card ID", catalog: definitionCatalogForTest{}, cardID: "  ", wantErrPart: "cardID cannot be empty"},
		{name: "unknown card ID", catalog: definitionCatalogForTest{}, cardID: "missing-card", wantErrPart: "error looking up card"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			card, err := ResolveCardDefinition(testCase.catalog, testCase.cardID)
			if err == nil || !strings.Contains(err.Error(), testCase.wantErrPart) {
				t.Fatalf("ResolveCardDefinition() error = %v; want containing %q", err, testCase.wantErrPart)
			}
			if !reflect.DeepEqual(card, gamecards.Card{}) {
				t.Fatalf("ResolveCardDefinition() card = %#v; want zero value", card)
			}
		})
	}
}
