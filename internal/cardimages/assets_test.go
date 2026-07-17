package cardimages

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestInstallBundledCardBack verifies the embedded asset is installed byte-for-byte.
func TestInstallBundledCardBack(t *testing.T) {
	directory := t.TempDir()
	if err := InstallBundledCardBack(directory); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(filepath.Join(directory, CardBackFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	configuration, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatalf("decode bundled card back: %v", err)
	}
	if configuration.Width != 896 || configuration.Height != 1268 {
		t.Fatalf("card-back size = %dx%d", configuration.Width, configuration.Height)
	}
}
