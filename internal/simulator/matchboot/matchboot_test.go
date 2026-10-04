package matchboot

import (
	"path/filepath"
	"testing"
)

func TestNewPrototypeMatchFromRepoData(t *testing.T) {
	repository, err := LoadCatalog(filepath.Join("..", "..", "..", "data", "cards.json"))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	match, err := NewPrototypeMatch(repository, NewSeed())
	if err != nil {
		t.Fatalf("NewPrototypeMatch() error = %v", err)
	}
	if match == nil {
		t.Fatal("NewPrototypeMatch() returned nil match")
	}
}
