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
		{name: "rare printing", cardName: "Arthur Rare", cardType: "Caster", level: "1", wantName: "Arthur"},
		{name: "super rare level after decoration", cardName: "Arthur Super lv2 Rare", cardType: "Caster", level: "2", wantName: "Arthur"},
		{name: "super rare level before decoration", cardName: "Passion Wing lv2 Super Rare", cardType: "Caster", level: "2", wantName: "Passion Wing"},
		{name: "level two super rare without level text", cardName: "Arthur Super Rare", cardType: "Caster", level: "2", wantName: "Arthur"},
		{name: "hyper rare printing", cardName: "Passion Wing Hyper Rare", cardType: "Caster", level: "1", wantName: "Passion Wing"},
		{name: "alternate art", cardName: "Harpy Alternate Art", cardType: "Servant", level: "4", wantName: "Harpy"},
		{name: "sample printing", cardName: "Afternoon Ramie Sample", cardType: "Caster", level: "1", wantName: "Afternoon Ramie"},
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

func TestNormalizeSourceNameRemovesUpstreamPrintingLabels(t *testing.T) {
	tests := map[string]string{
		"Arthur Lv2":                    "Arthur",
		"Arthur Rare":                   "Arthur",
		"Arthur Super lv2 Rare":         "Arthur",
		"Passion Wing lv2 Super Rare":   "Passion Wing",
		"Passion Wing Hyper Rare":       "Passion Wing",
		"Afternoon Ramie Alternate Art": "Afternoon Ramie",
		"Afternoon Ramie Sample":        "Afternoon Ramie",
	}

	for source, want := range tests {
		if got := NormalizeSourceName(source); got != want {
			t.Errorf("NormalizeSourceName(%q) = %q; want %q", source, got, want)
		}
	}
}
