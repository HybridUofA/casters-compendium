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
