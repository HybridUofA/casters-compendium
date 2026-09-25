package releasenotes

import (
	"strings"
	"testing"
)

func TestAllReturnsBundledNotesNewestFirst(t *testing.T) {
	notes, err := All()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) < 1 {
		t.Fatal("All() returned no bundled release notes")
	}
	if notes[0].Version != "v0.2.0-rc.3" {
		t.Fatalf("newest bundled version = %q, want v0.2.0-rc.3", notes[0].Version)
	}
	if !strings.Contains(notes[0].Markdown, "Caster's Compendium v0.2.0-rc.3") {
		t.Fatal("newest bundled note does not contain its release heading")
	}
}

func TestCompareVersionsUsesSemanticPrereleaseOrder(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{left: "v0.2.1", right: "v0.2.1-rc.1", want: 1},
		{left: "v0.2.1-rc.10", right: "v0.2.1-rc.2", want: 1},
		{left: "v0.2.1-rc.1", right: "v0.2.0", want: 1},
		{left: "v0.1.6-hotfix.1", right: "v0.1.6-hotfix.1", want: 0},
	}
	for _, test := range tests {
		if got := compareVersions(test.left, test.right); got != test.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}
