package deckexport

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type TTSInstallPaths struct {
	Root                 string
	ImageDirectory       string
	SavedObjectDirectory string
	MainFacePath         string
	SideFacePath         string
	BackPath             string
	JSONPath             string
}

var unsafeTTSFilenameChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

func safeTTSFileName(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeTTSFilenameChars.ReplaceAllString(name, "-")
	name = strings.TrimRight(name, ". ")
	if name == "" {
		return "Deck"
	}
	return name
}

func pathPlanner(
	root string,
	deckName string,
	hasSideboard bool,
) (TTSInstallPaths, error) {
	if strings.TrimSpace(root) == "" {
		return TTSInstallPaths{}, fmt.Errorf("root path cannot be empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return TTSInstallPaths{}, fmt.Errorf("make TTS root path absolute: %w", err)
	}
	imageDirectory := filepath.Join(absRoot, "Mods", "Images", "CastersCompendium")
	savedObjectDirectory := filepath.Join(absRoot, "Saves", "Saved Objects", "The Caster Chronicles")
	safeName := safeTTSFileName(deckName)
	var sideFacePath string
	mainFacePath := filepath.Join(imageDirectory, safeName+"-main.png")
	if hasSideboard {
		sideFacePath = filepath.Join(imageDirectory, safeName+"-side.png")
	}
	backPath := filepath.Join(imageDirectory, safeName+"-back.png")
	jsonPath := filepath.Join(savedObjectDirectory, safeName+".json")
	var installPaths = TTSInstallPaths{
		Root:                 absRoot,
		ImageDirectory:       imageDirectory,
		SavedObjectDirectory: savedObjectDirectory,
		MainFacePath:         mainFacePath,
		SideFacePath:         sideFacePath,
		BackPath:             backPath,
		JSONPath:             jsonPath,
	}
	return installPaths, nil
}

func prepareTTSDirectories(paths TTSInstallPaths) error {
	if strings.TrimSpace(paths.Root) == "" {
		return fmt.Errorf("TTS root cannot be empty")
	}
	if !filepath.IsAbs(paths.Root) {
		return fmt.Errorf("TTS root must be absolute: %q", paths.Root)
	}
	info, err := os.Stat(paths.Root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("TTS root not found: %q", paths.Root)
		}
		return fmt.Errorf("inspect TTS root %q: %w", paths.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("TTS root is not a directory: %q", paths.Root)
	}
	mods, err := os.Stat(filepath.Join(paths.Root, "Mods"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("TTS Mods directory does not exist: %q", filepath.Join(paths.Root, "Mods"))
		}
		return fmt.Errorf("inspect TTS Mods directory %q: %w", filepath.Join(paths.Root, "Mods"), err)
	}
	if !mods.IsDir() {
		return fmt.Errorf("TTS Mods directory is not a directory: %q", filepath.Join(paths.Root, "Mods"))
	}
	saves, err := os.Stat(filepath.Join(paths.Root, "Saves"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("TTS Saves directory does not exist: %q", filepath.Join(paths.Root, "Saves"))
		}
		return fmt.Errorf("inspect TTS Saves directory %q: %w", filepath.Join(paths.Root, "Saves"), err)
	}
	if !saves.IsDir() {
		return fmt.Errorf("TTS Saves directory is not a directory: %q", filepath.Join(paths.Root, "Saves"))
	}

	expectedImageDirectory := filepath.Join(paths.Root, "Mods", "Images", "CastersCompendium")
	expectedSavedObjectDirectory := filepath.Join(paths.Root, "Saves", "Saved Objects", "The Caster Chronicles")
	if filepath.Clean(paths.ImageDirectory) != expectedImageDirectory || filepath.Clean(paths.SavedObjectDirectory) != expectedSavedObjectDirectory {
		return fmt.Errorf("supplied path not formatted as expected")
	}
	err = os.MkdirAll(paths.ImageDirectory, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}
	err = os.MkdirAll(paths.SavedObjectDirectory, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create saved object directory: %w", err)
	}

	return nil
}

func writeTTSFileAtomically(
	destination string,
	writeContents func(io.Writer) error,
) (err error) {
	if strings.TrimSpace(destination) == "" {
		return fmt.Errorf("destination cannot be empty")
	}
	if writeContents == nil {
		return fmt.Errorf("write contents cannot be nil")
	}
	directory := filepath.Dir(destination)
	temp, err := os.CreateTemp(directory, ".casters-compendium-*")
	if err != nil {
		return fmt.Errorf("create temporary TTS file: %w", err)
	}
	tempPath := temp.Name()
	closed := false
	defer os.Remove(tempPath)
	defer func() {
		if !closed {
			temp.Close()
		}
	}()

	if err := writeContents(temp); err != nil {
		return fmt.Errorf("write temporary TTS file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary TTS file: %w", err)
	}
	closed = true

	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("install TTS file %q: %w", destination, err)
	}
	return nil
}

func copyTTSCardBack(
	writer io.Writer,
	source io.Reader,
) error {
	if writer == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if source == nil {
		return fmt.Errorf("source cannot be nil")
	}

	_, err := io.Copy(writer, source)
	if err != nil {
		return fmt.Errorf("failed to copy from source: %w", err)
	}
	return nil
}

// InstallTTSDeck builds and installs one deck as a local Tabletop Simulator
// saved object. The supplied card-back reader keeps the installer independent
// of the UI's default-versus-custom back selection.
func InstallTTSDeck(
	root string,
	deck *decks.Deck,
	repository decks.CardCatalog,
	cardImageDirectory string,
	cardBack io.Reader,
) (TTSInstallPaths, error) {
	if deck == nil {
		return TTSInstallPaths{}, fmt.Errorf("deck cannot be nil")
	}
	if repository == nil {
		return TTSInstallPaths{}, fmt.Errorf("repository cannot be nil")
	}
	if strings.TrimSpace(cardImageDirectory) == "" {
		return TTSInstallPaths{}, fmt.Errorf("card image directory cannot be empty")
	}
	if cardBack == nil {
		return TTSInstallPaths{}, fmt.Errorf("card back cannot be nil")
	}

	paths, err := pathPlanner(root, deck.Name, deck.SideTotal() > 0)
	if err != nil {
		return TTSInstallPaths{}, fmt.Errorf("plan TTS installation: %w", err)
	}
	savedObject, mainSheetIDs, sideSheetIDs, err := buildSavedObject(
		deck,
		paths.MainFacePath,
		paths.SideFacePath,
		paths.BackPath,
		repository,
	)
	if err != nil {
		return TTSInstallPaths{}, fmt.Errorf("build TTS saved object: %w", err)
	}
	if err := installTTSDeck(
		paths,
		mainSheetIDs,
		sideSheetIDs,
		cardImageDirectory,
		cardBack,
		savedObject,
	); err != nil {
		return TTSInstallPaths{}, fmt.Errorf("install TTS deck: %w", err)
	}
	return paths, nil
}

func installTTSDeck(
	paths TTSInstallPaths,
	mainSheetIDs []string,
	sideSheetIDs []string,
	cardImageDirectory string,
	cardBack io.Reader,
	savedObject SavedObject,
) error {
	if err := prepareTTSDirectories(paths); err != nil {
		return fmt.Errorf("prepare TTS installation: %w", err)
	}
	err := writeTTSFileAtomically(
		paths.MainFacePath,
		func(writer io.Writer) error {
			return writeTTSFaceSheet(
				writer,
				mainSheetIDs,
				cardImageDirectory,
			)
		},
	)
	if err != nil {
		return fmt.Errorf("install main TTS face sheet: %w", err)
	}

	if len(sideSheetIDs) > 0 {
		err = writeTTSFileAtomically(
			paths.SideFacePath,
			func(writer io.Writer) error {
				return writeTTSFaceSheet(
					writer,
					sideSheetIDs,
					cardImageDirectory,
				)
			},
		)
		if err != nil {
			return fmt.Errorf("install side TTS face sheet: %w", err)
		}
	}
	err = writeTTSFileAtomically(paths.BackPath,
		func(writer io.Writer) error {
			return copyTTSCardBack(writer, cardBack)
		},
	)
	if err != nil {
		return fmt.Errorf("install TTS card back: %w", err)
	}

	err = writeTTSFileAtomically(
		paths.JSONPath,
		func(writer io.Writer) error {
			return writeSavedObjectJSON(writer, savedObject)
		},
	)
	if err != nil {
		return fmt.Errorf("install TTS saved object: %w", err)
	}

	return nil
}
