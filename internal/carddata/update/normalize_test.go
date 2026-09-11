package cardupdate

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

func TestFromSpeedroboNormalizesDuplicatedCasterLevelSuffix(t *testing.T) {
	tests := []struct {
		name     string
		cardName string
		cardType string
		level    string
		wantName string
	}{
		{name: "canonical suffix", cardName: "Arthur Lv2", cardType: "Caster", level: "2", wantName: "Arthur"},
		{name: "lowercase suffix", cardName: "Carella lv2", cardType: "caster", level: "2", wantName: "Carella"},
		{name: "spaced suffix", cardName: "Example LV 3", cardType: " CASTER ", level: "3", wantName: "Example"},
		{name: "level mismatch", cardName: "Example Lv2", cardType: "Caster", level: "3", wantName: "Example Lv2"},
		{name: "level one name", cardName: "Example Lv2", cardType: "Caster", level: "1", wantName: "Example Lv2"},
		{name: "non-Caster", cardName: "Example Lv2", cardType: "Servant", level: "2", wantName: "Example Lv2"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			card, err := FromSpeedrobo(speedrobo.CardDetail{
				ID:            "test-card",
				CardKey:       testCase.cardName,
				ExpansionName: "Test",
				IsPlaytesting: "0",
				Fields: []speedrobo.CardField{
					{Label: "Name", Value: testCase.cardName},
					{Label: "Type", Value: testCase.cardType},
					{Label: "Cost/Lv", Value: testCase.level},
				},
			})
			if err != nil {
				t.Fatalf("FromSpeedrobo() error = %v", err)
			}
			if card.Name != testCase.wantName {
				t.Fatalf("normalized name = %q; want %q", card.Name, testCase.wantName)
			}
		})
	}
}
