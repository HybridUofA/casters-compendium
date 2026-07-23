package deckexport

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	dataassets "github.com/HybridUofA/casters-compendium/data"
	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// TestSafeTTSFileName verifies readable characters and internal spaces remain
// intact while cross-platform forbidden characters are replaced.
func TestSafeTTSFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "readable", input: "Luna Aqua Control", want: "Luna Aqua Control"},
		{name: "trim outer whitespace", input: "  Luna Aqua  ", want: "Luna Aqua"},
		{
			name:  "replace forbidden characters",
			input: `Luna/Aqua\Control:Test*Deck?"One"<Two>|Three`,
			want:  "Luna-Aqua-Control-Test-Deck--One--Two--Three",
		},
		{name: "replace control characters", input: "Deck\nName\tTest", want: "Deck-Name-Test"},
		{name: "trim trailing dots and spaces", input: "Deck...   ", want: "Deck"},
		{name: "blank fallback", input: "   ", want: "Deck"},
		{name: "punctuation fallback", input: "...   ", want: "Deck"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := safeTTSFileName(test.input); got != test.want {
				t.Fatalf("safeTTSFileName(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

// TestPathPlannerWithSideboard verifies all planned paths share the absolute
// TTS root and use safe filenames in their correct directory branches.
func TestPathPlannerWithSideboard(t *testing.T) {
	root := t.TempDir()

	got, err := pathPlanner(root, "Luna/Aqua: Control?", true)
	if err != nil {
		t.Fatal(err)
	}
	imageDirectory := filepath.Join(root, "Mods", "Images", "CastersCompendium")
	savedObjectDirectory := filepath.Join(
		root,
		"Saves",
		"Saved Objects",
		"The Caster Chronicles",
	)
	want := TTSInstallPaths{
		Root:                 root,
		ImageDirectory:       imageDirectory,
		SavedObjectDirectory: savedObjectDirectory,
		MainFacePath: filepath.Join(
			imageDirectory,
			"Luna-Aqua- Control--main.png",
		),
		SideFacePath: filepath.Join(
			imageDirectory,
			"Luna-Aqua- Control--side.png",
		),
		BackPath: filepath.Join(
			imageDirectory,
			"Luna-Aqua- Control--back.png",
		),
		JSONPath: filepath.Join(
			savedObjectDirectory,
			"Luna-Aqua- Control-.json",
		),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pathPlanner() = %#v, want %#v", got, want)
	}
}

// TestPathPlannerWithoutSideboard verifies the optional side path remains
// empty while every required output is still planned.
func TestPathPlannerWithoutSideboard(t *testing.T) {
	root := t.TempDir()

	got, err := pathPlanner(root, "Main Only", false)
	if err != nil {
		t.Fatal(err)
	}
	if got.SideFacePath != "" {
		t.Fatalf("SideFacePath = %q, want empty", got.SideFacePath)
	}
	if got.MainFacePath == "" || got.BackPath == "" || got.JSONPath == "" {
		t.Fatalf("required paths are incomplete: %#v", got)
	}
}

// TestPathPlannerMakesRootAbsolute verifies relative inputs do not leak into
// JSON asset references or installation destinations.
func TestPathPlannerMakesRootAbsolute(t *testing.T) {
	relativeRoot := filepath.Join("testdata", "tts-root")
	wantRoot, err := filepath.Abs(relativeRoot)
	if err != nil {
		t.Fatal(err)
	}

	got, err := pathPlanner(relativeRoot, "Deck", false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != wantRoot || !filepath.IsAbs(got.Root) {
		t.Fatalf("Root = %q, want absolute %q", got.Root, wantRoot)
	}
	for _, path := range []string{
		got.ImageDirectory,
		got.SavedObjectDirectory,
		got.MainFacePath,
		got.BackPath,
		got.JSONPath,
	} {
		if !filepath.IsAbs(path) || !strings.HasPrefix(path, wantRoot) {
			t.Fatalf("planned path %q is not beneath root %q", path, wantRoot)
		}
	}
}

// TestPathPlannerRejectsEmptyRoot verifies blank roots return no partial plan.
func TestPathPlannerRejectsEmptyRoot(t *testing.T) {
	for _, root := range []string{"", "   "} {
		t.Run("root_"+strings.ReplaceAll(root, " ", "_"), func(t *testing.T) {
			got, err := pathPlanner(root, "Deck", true)
			if err == nil || !strings.Contains(err.Error(), "root path cannot be empty") {
				t.Fatalf("pathPlanner() error = %v", err)
			}
			if got != (TTSInstallPaths{}) {
				t.Fatalf("pathPlanner() = %#v, want zero value", got)
			}
		})
	}
}

// TestPrepareTTSDirectories creates both application-owned destinations beneath
// an existing TTS layout and remains safe to call repeatedly.
func TestPrepareTTSDirectories(t *testing.T) {
	root := newTestTTSRoot(t)
	paths, err := pathPlanner(root, "Luna Aqua Control", true)
	if err != nil {
		t.Fatal(err)
	}

	for run := 1; run <= 2; run++ {
		if err := prepareTTSDirectories(paths); err != nil {
			t.Fatalf("prepare run %d: %v", run, err)
		}
		for _, directory := range []string{
			paths.ImageDirectory,
			paths.SavedObjectDirectory,
		} {
			info, err := os.Stat(directory)
			if err != nil {
				t.Fatalf("stat prepared directory %q: %v", directory, err)
			}
			if !info.IsDir() {
				t.Fatalf("prepared path %q is not a directory", directory)
			}
		}
	}
}

// TestPrepareTTSDirectoriesRejectsInvalidRoot verifies malformed, missing, and
// nondirectory roots fail before any destination is created.
func TestPrepareTTSDirectoriesRejectsInvalidRoot(t *testing.T) {
	parent := t.TempDir()
	rootFile := filepath.Join(parent, "tts-file")
	if err := os.WriteFile(rootFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(parent, "missing")

	tests := []struct {
		name      string
		paths     TTSInstallPaths
		wantError string
	}{
		{name: "blank", paths: TTSInstallPaths{}, wantError: "root cannot be empty"},
		{
			name:      "relative",
			paths:     TTSInstallPaths{Root: "relative/tts"},
			wantError: "must be absolute",
		},
		{
			name:      "missing",
			paths:     TTSInstallPaths{Root: missingRoot},
			wantError: "root not found",
		},
		{
			name:      "file",
			paths:     TTSInstallPaths{Root: rootFile},
			wantError: "not a directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := prepareTTSDirectories(test.paths)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("prepareTTSDirectories() error = %v", err)
			}
		})
	}
}

// TestPrepareTTSDirectoriesRequiresTTSParents verifies the selected root
// already contains directory-shaped Mods and Saves entries.
func TestPrepareTTSDirectoriesRequiresTTSParents(t *testing.T) {
	tests := []struct {
		name      string
		setupRoot func(t *testing.T, root string)
		wantError string
	}{
		{
			name: "missing Mods",
			setupRoot: func(t *testing.T, root string) {
				t.Helper()
				mkdirTestDirectory(t, filepath.Join(root, "Saves"))
			},
			wantError: "Mods directory does not exist",
		},
		{
			name: "Mods is a file",
			setupRoot: func(t *testing.T, root string) {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "Mods"))
				mkdirTestDirectory(t, filepath.Join(root, "Saves"))
			},
			wantError: "Mods directory is not a directory",
		},
		{
			name: "missing Saves",
			setupRoot: func(t *testing.T, root string) {
				t.Helper()
				mkdirTestDirectory(t, filepath.Join(root, "Mods"))
			},
			wantError: "Saves directory does not exist",
		},
		{
			name: "Saves is a file",
			setupRoot: func(t *testing.T, root string) {
				t.Helper()
				mkdirTestDirectory(t, filepath.Join(root, "Mods"))
				writeTestFile(t, filepath.Join(root, "Saves"))
			},
			wantError: "Saves directory is not a directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			test.setupRoot(t, root)
			paths, err := pathPlanner(root, "Deck", true)
			if err != nil {
				t.Fatal(err)
			}

			err = prepareTTSDirectories(paths)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("prepareTTSDirectories() error = %v", err)
			}
		})
	}
}

// TestPrepareTTSDirectoriesRejectsTamperedDestinations verifies caller-supplied
// output directories cannot escape or alter the fixed TTS layout.
func TestPrepareTTSDirectoriesRejectsTamperedDestinations(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(paths *TTSInstallPaths)
	}{
		{
			name: "image directory",
			tamper: func(paths *TTSInstallPaths) {
				paths.ImageDirectory = filepath.Join(paths.Root, "elsewhere", "images")
			},
		},
		{
			name: "saved object directory",
			tamper: func(paths *TTSInstallPaths) {
				paths.SavedObjectDirectory = filepath.Join(paths.Root, "elsewhere", "objects")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newTestTTSRoot(t)
			paths, err := pathPlanner(root, "Deck", true)
			if err != nil {
				t.Fatal(err)
			}
			test.tamper(&paths)

			err = prepareTTSDirectories(paths)
			if err == nil || !strings.Contains(err.Error(), "not formatted as expected") {
				t.Fatalf("prepareTTSDirectories() error = %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(root, "elsewhere")); !os.IsNotExist(statErr) {
				t.Fatalf("tampered destination was created; stat error = %v", statErr)
			}
		})
	}
}

// TestPrepareTTSDirectoriesReportsCreationFailures verifies errors from each
// application-owned directory are returned with their purpose.
func TestPrepareTTSDirectoriesReportsCreationFailures(t *testing.T) {
	t.Run("image directory", func(t *testing.T) {
		root := newTestTTSRoot(t)
		writeTestFile(t, filepath.Join(root, "Mods", "Images"))
		paths, err := pathPlanner(root, "Deck", false)
		if err != nil {
			t.Fatal(err)
		}

		err = prepareTTSDirectories(paths)
		if err == nil || !strings.Contains(err.Error(), "create image directory") {
			t.Fatalf("prepareTTSDirectories() error = %v", err)
		}
	})

	t.Run("saved object directory", func(t *testing.T) {
		root := newTestTTSRoot(t)
		writeTestFile(t, filepath.Join(root, "Saves", "Saved Objects"))
		paths, err := pathPlanner(root, "Deck", false)
		if err != nil {
			t.Fatal(err)
		}

		err = prepareTTSDirectories(paths)
		if err == nil || !strings.Contains(err.Error(), "create saved object directory") {
			t.Fatalf("prepareTTSDirectories() error = %v", err)
		}
	})
}

func TestWriteTTSFile(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "deck.json")
	want := "first part\nsecond part\n"

	err := writeTTSFileAtomically(destination, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, "first part\n"); err != nil {
			return err
		}
		_, err := io.WriteString(writer, "second part\n")
		return err
	})
	if err != nil {
		t.Fatalf("writeTTSFileAtomically() error = %v", err)
	}

	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if string(got) != want {
		t.Fatalf("installed contents = %q, want %q", got, want)
	}
	assertNoTTSTemporaryFiles(t, directory)
}

func TestWriteTTSFileRejectsInvalidArguments(t *testing.T) {
	tests := []struct {
		name        string
		destination string
		write       func(io.Writer) error
		wantError   string
	}{
		{
			name:        "empty destination",
			destination: "",
			write:       func(io.Writer) error { return nil },
			wantError:   "destination cannot be empty",
		},
		{
			name:        "whitespace destination",
			destination: " \t\n",
			write:       func(io.Writer) error { return nil },
			wantError:   "destination cannot be empty",
		},
		{
			name:        "nil callback",
			destination: filepath.Join(t.TempDir(), "deck.json"),
			write:       nil,
			wantError:   "write contents cannot be nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := writeTTSFileAtomically(test.destination, test.write)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("writeTTSFileAtomically() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestWriteTTSFileReportsMissingDestinationDirectory(t *testing.T) {
	root := t.TempDir()
	missingDirectory := filepath.Join(root, "missing")
	destination := filepath.Join(missingDirectory, "deck.json")
	called := false

	err := writeTTSFileAtomically(destination, func(io.Writer) error {
		called = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "create temporary TTS file") {
		t.Fatalf("writeTTSFileAtomically() error = %v", err)
	}
	if called {
		t.Fatal("write callback was called after temporary-file creation failed")
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination exists after failure; stat error = %v", statErr)
	}
}

func TestWriteTTSFilePreservesDestinationWhenCallbackFails(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "deck.json")
	original := []byte("known-good export")
	if err := os.WriteFile(destination, original, 0o644); err != nil {
		t.Fatal(err)
	}
	writeErr := errors.New("deliberate write failure")

	err := writeTTSFileAtomically(destination, func(writer io.Writer) error {
		if _, err := io.WriteString(writer, "incomplete replacement"); err != nil {
			return err
		}
		return writeErr
	})
	if !errors.Is(err, writeErr) {
		t.Fatalf("writeTTSFileAtomically() error = %v, want wrapping %v", err, writeErr)
	}

	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("destination contents = %q, want preserved %q", got, original)
	}
	assertNoTTSTemporaryFiles(t, directory)
}

func TestWriteTTSFileCleansUpAfterRenameFailure(t *testing.T) {
	directory := t.TempDir()
	// Renaming a file onto a non-empty directory fails consistently on the
	// supported desktop platforms without depending on overwrite semantics.
	destination := filepath.Join(directory, "occupied")
	mkdirTestDirectory(t, destination)
	writeTestFile(t, filepath.Join(destination, "keep.txt"))

	err := writeTTSFileAtomically(destination, func(writer io.Writer) error {
		_, err := io.WriteString(writer, "complete temporary contents")
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "install TTS file") {
		t.Fatalf("writeTTSFileAtomically() error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(destination, "keep.txt")); statErr != nil {
		t.Fatalf("existing destination was damaged: %v", statErr)
	}
	assertNoTTSTemporaryFiles(t, directory)
}

func TestCopyTTSCardBack(t *testing.T) {
	want := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0xff}
	var destination bytes.Buffer

	if err := copyTTSCardBack(&destination, bytes.NewReader(want)); err != nil {
		t.Fatalf("copyTTSCardBack() error = %v", err)
	}
	if !bytes.Equal(destination.Bytes(), want) {
		t.Fatalf("copied bytes = %v, want %v", destination.Bytes(), want)
	}
}

func TestCopyTTSCardBackAcceptsEmptySource(t *testing.T) {
	var destination bytes.Buffer

	if err := copyTTSCardBack(&destination, strings.NewReader("")); err != nil {
		t.Fatalf("copyTTSCardBack() error = %v", err)
	}
	if destination.Len() != 0 {
		t.Fatalf("copied %d bytes from empty source", destination.Len())
	}
}

func TestCopyTTSCardBackRejectsNilArguments(t *testing.T) {
	tests := []struct {
		name      string
		writer    io.Writer
		source    io.Reader
		wantError string
	}{
		{
			name:      "nil writer",
			writer:    nil,
			source:    strings.NewReader("back"),
			wantError: "writer cannot be nil",
		},
		{
			name:      "nil source",
			writer:    io.Discard,
			source:    nil,
			wantError: "source cannot be nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := copyTTSCardBack(test.writer, test.source)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("copyTTSCardBack() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestCopyTTSCardBackReportsCopyFailures(t *testing.T) {
	readErr := errors.New("deliberate read failure")
	writeErr := errors.New("deliberate write failure")
	tests := []struct {
		name   string
		writer io.Writer
		source io.Reader
		want   error
	}{
		{
			name:   "source read",
			writer: io.Discard,
			source: errorTTSReader{err: readErr},
			want:   readErr,
		},
		{
			name:   "destination write",
			writer: errorTTSWriter{err: writeErr},
			source: strings.NewReader("back"),
			want:   writeErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := copyTTSCardBack(test.writer, test.source)
			if !errors.Is(err, test.want) {
				t.Fatalf("copyTTSCardBack() error = %v, want wrapping %v", err, test.want)
			}
		})
	}
}

func TestInstallTTSDeckMainOnly(t *testing.T) {
	root := newTestTTSRoot(t)
	paths, err := pathPlanner(root, "Luna Control", false)
	if err != nil {
		t.Fatal(err)
	}
	cardImageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)
	savedObject := testTTSSavedObject("Luna Control")

	err = installTTSDeck(
		paths,
		[]string{"main"},
		nil,
		cardImageDirectory,
		bytes.NewReader(dataassets.CardBackPNG),
		savedObject,
	)
	if err != nil {
		t.Fatalf("installTTSDeck() error = %v", err)
	}

	assertPNGFile(t, paths.MainFacePath)
	assertFileBytes(t, paths.BackPath, dataassets.CardBackPNG)
	assertSavedObjectFile(t, paths.JSONPath, savedObject)
	if paths.SideFacePath != "" {
		t.Fatalf("SideFacePath = %q for main-only export", paths.SideFacePath)
	}
	matches, err := filepath.Glob(filepath.Join(paths.ImageDirectory, "*-side.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected side sheets installed: %v", matches)
	}
}

func TestInstallTTSDeckMainAndSideboard(t *testing.T) {
	root := newTestTTSRoot(t)
	paths, err := pathPlanner(root, "Luna Control", true)
	if err != nil {
		t.Fatal(err)
	}
	cardImageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)
	writeSolidPNG(t, filepath.Join(cardImageDirectory, "side.png"), deckImageBackground)
	savedObject := testTTSSavedObject("Luna Control")

	err = installTTSDeck(
		paths,
		[]string{"main"},
		[]string{"side"},
		cardImageDirectory,
		bytes.NewReader(dataassets.CardBackPNG),
		savedObject,
	)
	if err != nil {
		t.Fatalf("installTTSDeck() error = %v", err)
	}

	assertPNGFile(t, paths.MainFacePath)
	assertPNGFile(t, paths.SideFacePath)
	assertFileBytes(t, paths.BackPath, dataassets.CardBackPNG)
	assertSavedObjectFile(t, paths.JSONPath, savedObject)
}

func TestInstallTTSDeckStopsAfterStageFailure(t *testing.T) {
	t.Run("directory preparation", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "missing")
		paths, err := pathPlanner(root, "Deck", false)
		if err != nil {
			t.Fatal(err)
		}

		err = installTTSDeck(
			paths,
			[]string{"main"},
			nil,
			t.TempDir(),
			bytes.NewReader(dataassets.CardBackPNG),
			testTTSSavedObject("Deck"),
		)
		if err == nil || !strings.Contains(err.Error(), "prepare TTS installation") {
			t.Fatalf("installTTSDeck() error = %v", err)
		}
		assertPathDoesNotExist(t, paths.MainFacePath)
		assertPathDoesNotExist(t, paths.BackPath)
		assertPathDoesNotExist(t, paths.JSONPath)
	})

	t.Run("main face sheet", func(t *testing.T) {
		paths, err := pathPlanner(newTestTTSRoot(t), "Deck", false)
		if err != nil {
			t.Fatal(err)
		}

		err = installTTSDeck(
			paths,
			[]string{"missing-main"},
			nil,
			t.TempDir(),
			bytes.NewReader(dataassets.CardBackPNG),
			testTTSSavedObject("Deck"),
		)
		if err == nil || !strings.Contains(err.Error(), "install main TTS face sheet") {
			t.Fatalf("installTTSDeck() error = %v", err)
		}
		assertPathDoesNotExist(t, paths.MainFacePath)
		assertPathDoesNotExist(t, paths.BackPath)
		assertPathDoesNotExist(t, paths.JSONPath)
		assertNoTTSTemporaryFiles(t, paths.ImageDirectory)
	})

	t.Run("side face sheet", func(t *testing.T) {
		paths, err := pathPlanner(newTestTTSRoot(t), "Deck", true)
		if err != nil {
			t.Fatal(err)
		}
		cardImageDirectory := t.TempDir()
		writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)

		err = installTTSDeck(
			paths,
			[]string{"main"},
			[]string{"missing-side"},
			cardImageDirectory,
			bytes.NewReader(dataassets.CardBackPNG),
			testTTSSavedObject("Deck"),
		)
		if err == nil || !strings.Contains(err.Error(), "install side TTS face sheet") {
			t.Fatalf("installTTSDeck() error = %v", err)
		}
		assertPNGFile(t, paths.MainFacePath)
		assertPathDoesNotExist(t, paths.SideFacePath)
		assertPathDoesNotExist(t, paths.BackPath)
		assertPathDoesNotExist(t, paths.JSONPath)
		assertNoTTSTemporaryFiles(t, paths.ImageDirectory)
	})

	t.Run("card back", func(t *testing.T) {
		paths, err := pathPlanner(newTestTTSRoot(t), "Deck", false)
		if err != nil {
			t.Fatal(err)
		}
		cardImageDirectory := t.TempDir()
		writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)
		copyErr := errors.New("deliberate card-back read failure")

		err = installTTSDeck(
			paths,
			[]string{"main"},
			nil,
			cardImageDirectory,
			errorTTSReader{err: copyErr},
			testTTSSavedObject("Deck"),
		)
		if !errors.Is(err, copyErr) ||
			!strings.Contains(err.Error(), "install TTS card back") {
			t.Fatalf("installTTSDeck() error = %v, want wrapping %v", err, copyErr)
		}
		assertPNGFile(t, paths.MainFacePath)
		assertPathDoesNotExist(t, paths.BackPath)
		assertPathDoesNotExist(t, paths.JSONPath)
		assertNoTTSTemporaryFiles(t, paths.ImageDirectory)
	})

	t.Run("saved object JSON", func(t *testing.T) {
		paths, err := pathPlanner(newTestTTSRoot(t), "Deck", false)
		if err != nil {
			t.Fatal(err)
		}
		cardImageDirectory := t.TempDir()
		writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)

		err = installTTSDeck(
			paths,
			[]string{"main"},
			nil,
			cardImageDirectory,
			bytes.NewReader(dataassets.CardBackPNG),
			SavedObject{},
		)
		if err == nil || !strings.Contains(err.Error(), "install TTS saved object") {
			t.Fatalf("installTTSDeck() error = %v", err)
		}
		assertPNGFile(t, paths.MainFacePath)
		assertFileBytes(t, paths.BackPath, dataassets.CardBackPNG)
		assertPathDoesNotExist(t, paths.JSONPath)
		assertNoTTSTemporaryFiles(t, paths.SavedObjectDirectory)
	})
}

func TestInstallTTSDeckPublicCoordinator(t *testing.T) {
	root := newTestTTSRoot(t)
	cardImageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(cardImageDirectory, "main.png"), deckImageBackground)
	writeSolidPNG(t, filepath.Join(cardImageDirectory, "side.png"), deckImageBackground)
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "main", Name: "Main Card"},
		{ID: "side", Name: "Side Card"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck, err := decks.NewDeck("Coordinator Deck")
	if err != nil {
		t.Fatal(err)
	}
	if err := deck.AddCard(decks.MainZone, "main", 2); err != nil {
		t.Fatal(err)
	}
	if err := deck.AddCard(decks.SideZone, "side", 1); err != nil {
		t.Fatal(err)
	}
	deck.EnsureOrder()

	paths, err := InstallTTSDeck(
		root,
		deck,
		repository,
		cardImageDirectory,
		bytes.NewReader(dataassets.CardBackPNG),
	)
	if err != nil {
		t.Fatalf("InstallTTSDeck() error = %v", err)
	}

	assertPNGFile(t, paths.MainFacePath)
	assertPNGFile(t, paths.SideFacePath)
	assertFileBytes(t, paths.BackPath, dataassets.CardBackPNG)
	encoded, err := os.ReadFile(paths.JSONPath)
	if err != nil {
		t.Fatal(err)
	}
	var object SavedObject
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatal(err)
	}
	if object.SaveName != deck.Name || len(object.ObjectStates) != 2 {
		t.Fatalf("saved object identity = %q with %d states", object.SaveName, len(object.ObjectStates))
	}
	mainState := object.ObjectStates[0].CustomDeck["1"]
	if mainState.FaceURL != paths.MainFacePath || mainState.BackURL != paths.BackPath {
		t.Fatalf("main asset paths = %q / %q", mainState.FaceURL, mainState.BackURL)
	}
	sideState := object.ObjectStates[1].CustomDeck["2"]
	if sideState.FaceURL != paths.SideFacePath || sideState.BackURL != paths.BackPath {
		t.Fatalf("side asset paths = %q / %q", sideState.FaceURL, sideState.BackURL)
	}
}

func TestInstallTTSDeckPublicCoordinatorRejectsInvalidInputs(t *testing.T) {
	validDeck, err := decks.NewDeck("Deck")
	if err != nil {
		t.Fatal(err)
	}
	if err := validDeck.AddCard(decks.MainZone, "main", 1); err != nil {
		t.Fatal(err)
	}
	validDeck.EnsureOrder()
	validRepository, err := cards.NewRepository([]cards.Card{
		{ID: "main", Name: "Main Card"},
	})
	if err != nil {
		t.Fatal(err)
	}
	validRoot := newTestTTSRoot(t)
	validImages := t.TempDir()
	writeSolidPNG(t, filepath.Join(validImages, "main.png"), deckImageBackground)

	tests := []struct {
		name       string
		root       string
		deck       *decks.Deck
		repository decks.CardCatalog
		images     string
		back       io.Reader
		wantError  string
	}{
		{
			name:       "nil deck",
			root:       validRoot,
			repository: validRepository,
			images:     validImages,
			back:       strings.NewReader("back"),
			wantError:  "deck cannot be nil",
		},
		{
			name:      "nil repository",
			root:      validRoot,
			deck:      validDeck,
			images:    validImages,
			back:      strings.NewReader("back"),
			wantError: "repository cannot be nil",
		},
		{
			name:       "empty image directory",
			root:       validRoot,
			deck:       validDeck,
			repository: validRepository,
			images:     " \t",
			back:       strings.NewReader("back"),
			wantError:  "card image directory cannot be empty",
		},
		{
			name:       "nil card back",
			root:       validRoot,
			deck:       validDeck,
			repository: validRepository,
			images:     validImages,
			wantError:  "card back cannot be nil",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths, err := InstallTTSDeck(
				test.root,
				test.deck,
				test.repository,
				test.images,
				test.back,
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("InstallTTSDeck() error = %v, want containing %q", err, test.wantError)
			}
			if paths != (TTSInstallPaths{}) {
				t.Fatalf("InstallTTSDeck() paths = %#v after validation failure", paths)
			}
		})
	}
}

func testTTSSavedObject(name string) SavedObject {
	return SavedObject{
		SaveName: name,
		ObjectStates: []DeckObject{
			{
				Name:     "DeckCustom",
				Nickname: name,
			},
		},
	}
}

func assertPathDoesNotExist(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path %q exists or returned unexpected error: %v", path, err)
	}
}

func assertPNGFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := png.DecodeConfig(file); err != nil {
		t.Fatalf("decode PNG %q: %v", path, err)
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file %q differs from expected %d-byte contents", path, len(want))
	}
}

func assertSavedObjectFile(t *testing.T, path string, want SavedObject) {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got SavedObject
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode saved object %q: %v", path, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("saved object = %#v, want %#v", got, want)
	}
}

type errorTTSReader struct {
	err error
}

func (reader errorTTSReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type errorTTSWriter struct {
	err error
}

func (writer errorTTSWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func assertNoTTSTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".casters-compendium-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain after write: %v", matches)
	}
}

// newTestTTSRoot creates only the standard parent directories that must
// preexist before Caster's Compendium installs its own children.
func newTestTTSRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirTestDirectory(t, filepath.Join(root, "Mods"))
	mkdirTestDirectory(t, filepath.Join(root, "Saves"))
	return root
}

func mkdirTestDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
}
