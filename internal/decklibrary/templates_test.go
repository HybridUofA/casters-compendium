package decklibrary

import (
	"path/filepath"
	"testing"
	"time"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

func TestOfficialTemplatesBuildCompleteDecks(t *testing.T) {
	repository, err := cards.LoadFile(filepath.Join("..", "..", "data", "cards.json"))
	if err != nil {
		t.Fatal(err)
	}

	templates := OfficialTemplates()
	if len(templates) != 8 {
		t.Fatalf("OfficialTemplates() returned %d templates, want 8", len(templates))
	}
	seen := make(map[string]bool)
	for _, template := range templates {
		if seen[template.ID] {
			t.Fatalf("duplicate template ID %q", template.ID)
		}
		seen[template.ID] = true
		if template.SourceURL != officialDeckSourceURL {
			t.Errorf("%s source = %q", template.ID, template.SourceURL)
		}
		if _, err := time.Parse(time.DateOnly, template.SourceUpdated); err != nil {
			t.Errorf("%s source date: %v", template.ID, err)
		}

		deck, err := BuildOfficialTemplate(template.ID, repository)
		if err != nil {
			t.Errorf("build %s: %v", template.ID, err)
			continue
		}
		if deck.Name != template.Name {
			t.Errorf("%s name = %q, want %q", template.ID, deck.Name, template.Name)
		}
		if deck.MainTotal() != decks.MaxMainDeckCards {
			t.Errorf("%s total = %d", template.ID, deck.MainTotal())
		}
		if deck.SideTotal() != 0 {
			t.Errorf("%s side total = %d", template.ID, deck.SideTotal())
		}
	}
}

func TestBuildOfficialTemplateRejectsUnknownID(t *testing.T) {
	repository, err := cards.LoadFile(filepath.Join("..", "..", "data", "cards.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildOfficialTemplate("missing", repository); err == nil {
		t.Fatal("BuildOfficialTemplate() accepted an unknown ID")
	}
}
