package deckbuilder

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

func TestDiagnosticInformationContainsUsefulNonSensitiveContext(t *testing.T) {
	root := t.TempDir()
	paths := newApplicationPaths(root)
	if err := os.WriteFile(paths.CardDatabase, []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.SetupComplete, []byte("complete\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "1", Name: "Test Card"},
	})
	if err != nil {
		t.Fatal(err)
	}

	result := diagnosticInformation(paths, repository)
	for _, expected := range []string{
		"Application version: " + applicationVersion,
		"Go version: " + runtime.Version(),
		"Operating system: " + runtime.GOOS,
		"Architecture: " + runtime.GOARCH,
		"Card records: 1",
		"Setup complete: true",
		"Card database present: true",
		"Hosted catalog: " + hostedCatalogPointerURL,
	} {
		if !strings.Contains(result, expected) {
			t.Errorf("diagnostics missing %q:\n%s", expected, result)
		}
	}
	if strings.Contains(result, filepath.Base(root)) || strings.Contains(result, root) {
		t.Fatalf("diagnostics exposed local path %q:\n%s", root, result)
	}
}

func TestDiagnosticInformationHandlesMissingData(t *testing.T) {
	result := diagnosticInformation(newApplicationPaths(t.TempDir()), nil)
	for _, expected := range []string{
		"Card records: 0",
		"Setup complete: false",
		"Card database present: false",
	} {
		if !strings.Contains(result, expected) {
			t.Errorf("diagnostics missing %q:\n%s", expected, result)
		}
	}
}
