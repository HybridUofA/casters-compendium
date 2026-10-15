package effects

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type catalogCard struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Ability string `json:"ability"`
}

func TestMasterEffectsCoverEveryAbilityCard(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../.."))
	raw, err := os.ReadFile(filepath.Join(root, "data/cards.json"))
	if err != nil {
		t.Fatalf("read cards.json: %v", err)
	}
	var cards []catalogCard
	if err := json.Unmarshal(raw, &cards); err != nil {
		t.Fatalf("parse cards.json: %v", err)
	}

	byID := map[string][]EffectDecl{}
	for _, decl := range MasterEffects {
		byID[string(decl.CardID)] = append(byID[string(decl.CardID)], decl)
	}

	var missing []string
	var nilResolve []string
	for _, card := range cards {
		if card.Ability == "" {
			continue
		}
		decls, found := byID[card.ID]
		if !found || len(decls) == 0 {
			missing = append(missing, card.ID+" "+card.Name)
			continue
		}
		for _, decl := range decls {
			if decl.Resolve == nil && decl.Hook != HookNone {
				nilResolve = append(nilResolve, card.ID+" "+card.Name+" "+string(decl.Hook))
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("MasterEffects missing %d ability cards (first 10): %v", len(missing), missing[:min(10, len(missing))])
	}
	if len(nilResolve) > 0 {
		t.Fatalf("MasterEffects has Resolve:nil on active hooks (%d): %v", len(nilResolve), nilResolve[:min(10, len(nilResolve))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
