package cardimages

import (
	"fmt"
	"os"
	"path/filepath"

	dataassets "github.com/HybridUofA/casters-compendium/data"
)

// CardBackFileName is the stable filename used by Tabletop Simulator exports.
const CardBackFileName = "MTD-back-ver01.png"

// InstallBundledCardBack writes the embedded Tabletop Simulator card back into directory.
func InstallBundledCardBack(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create card-image directory: %w", err)
	}
	path := filepath.Join(directory, CardBackFileName)
	if err := os.WriteFile(path, dataassets.CardBackPNG, 0o644); err != nil {
		return fmt.Errorf("install bundled card back: %w", err)
	}
	return nil
}
