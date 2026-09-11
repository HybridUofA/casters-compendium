package catalog

import "testing"

func TestNewRepositoryNormalizesSourceOnlyCasterLevelSuffix(t *testing.T) {
	repository, err := NewRepository([]Card{
		{ID: "56", Name: "Arthur", Type: "Caster", CostLevel: "1"},
		{ID: "71", Name: "Arthur Lv2", Type: "Caster", CostLevel: "2"},
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	levelTwo, found := repository.FindByID("71")
	if !found {
		t.Fatal("normalized Level 2 definition was not indexed")
	}
	if levelTwo.Name != "Arthur" {
		t.Fatalf("Level 2 name = %q; want Arthur", levelTwo.Name)
	}
	if matches := repository.FindByName("Arthur"); len(matches) != 2 {
		t.Fatalf("FindByName(Arthur) returned %d definitions; want both levels", len(matches))
	}
}

func TestRepositoryResolvesLegacyIDToCanonicalCard(t *testing.T) {
	repository, err := NewRepository([]Card{{
		ID: "1225", LegacyIDs: []string{"235"}, Name: "Carella",
	}})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	card, found := repository.FindByID("235")
	if !found || card.ID != "1225" {
		t.Fatalf("FindByID(235) = %#v, %t; want canonical ID 1225", card, found)
	}
	if len(repository.All()) != 1 {
		t.Fatalf("All() included legacy alias as an additional card")
	}
}

func TestRepositoryRejectsLegacyIDCollision(t *testing.T) {
	_, err := NewRepository([]Card{
		{ID: "235", Name: "Historical"},
		{ID: "1225", LegacyIDs: []string{"235"}, Name: "Current"},
	})
	if err == nil {
		t.Fatal("NewRepository() accepted a legacy ID that conflicts with a canonical ID")
	}
}

func TestBundledRepositoryResolvesCarellaLegacyID(t *testing.T) {
	repository, err := LoadFile("../../../data/cards.json")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	wantAliases := map[string]string{
		"181": "1",
		"229": "1224",
		"235": "1225",
		"241": "1226",
		"242": "1227",
		"251": "1228",
		"252": "1229",
		"258": "1230",
		"264": "1231",
		"265": "1232",
	}
	for legacyID, canonicalID := range wantAliases {
		card, found := repository.FindByID(legacyID)
		if !found || card.ID != canonicalID {
			t.Errorf("legacy lookup %s = %#v, %t; want canonical ID %s", legacyID, card, found, canonicalID)
		}
	}
}
