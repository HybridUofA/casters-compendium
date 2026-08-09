package cards

import "testing"

func TestNormalizeDefinitionRemovesDuplicatedCasterLevelSuffix(t *testing.T) {
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
		{name: "level one", cardName: "Example Lv2", cardType: "Caster", level: "1", wantName: "Example Lv2"},
		{name: "non-Caster", cardName: "Example Lv2", cardType: "Servant", level: "2", wantName: "Example Lv2"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			card := NormalizeDefinition(Card{
				Name:      testCase.cardName,
				Type:      testCase.cardType,
				CostLevel: testCase.level,
			})
			if card.Name != testCase.wantName {
				t.Fatalf("normalized name = %q; want %q", card.Name, testCase.wantName)
			}
		})
	}
}
