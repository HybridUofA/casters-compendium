package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewApplicationPathsUsesApplicationRoot verifies all managed files stay under Fyne storage.
func TestNewApplicationPathsUsesApplicationRoot(t *testing.T) {
	root := t.TempDir()
	paths := newApplicationPaths(root)

	if paths.CardDatabase != filepath.Join(root, "cards.json") ||
		paths.CardListHash != filepath.Join(root, "cardlist.sha256") ||
		paths.Images != filepath.Join(root, "images") ||
		paths.Thumbnails != filepath.Join(root, "thumbnails") {
		t.Fatalf("unexpected application paths: %#v", paths)
	}
}

// TestLoadOrDownloadCardDatabaseUsesLocalDatabase verifies valid local data avoids network work.
func TestLoadOrDownloadCardDatabaseUsesLocalDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cards.json")
	data := []byte(`[{"id":"1","name":"Local Card","image_url":"local","expansion":"Test","card_number":"T-1"}]`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	progressCalled := false
	repository, err := loadOrDownloadCardDatabase(
		context.Background(),
		path,
		func(string, int, int) { progressCalled = true },
	)
	if err != nil {
		t.Fatal(err)
	}
	if progressCalled {
		t.Fatal("download progress was reported for an existing local database")
	}
	if len(repository.All()) != 1 {
		t.Fatalf("loaded %d cards", len(repository.All()))
	}
}

// TestWriteFileAtomically verifies existing content is replaced completely.
func TestWriteFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cards.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("file contents = %q", data)
	}
}

// TestRunSetupWorkersUsesBoundedConcurrency verifies the worker limit is reached but not exceeded.
func TestRunSetupWorkersUsesBoundedConcurrency(t *testing.T) {
	const workerCount = 3

	var active atomic.Int64
	var maximum atomic.Int64
	var started atomic.Int64
	ready := make(chan struct{}, workerCount)
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- runSetupWorkers(context.Background(), 12, workerCount, func(int) error {
			current := active.Add(1)
			defer active.Add(-1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}

			if started.Add(1) <= workerCount {
				ready <- struct{}{}
				<-release
			}
			return nil
		})
	}()

	for range workerCount {
		select {
		case <-ready:
		case <-time.After(time.Second):
			t.Fatal("workers did not start concurrently")
		}
	}
	if got := maximum.Load(); got != workerCount {
		t.Fatalf("maximum concurrent workers = %d, want %d", got, workerCount)
	}
	close(release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("workers did not finish")
	}
}

// TestRunSetupWorkersReturnsWorkError verifies worker failures propagate to the caller.
func TestRunSetupWorkersReturnsWorkError(t *testing.T) {
	want := errors.New("download failed")
	err := runSetupWorkers(context.Background(), 10, 3, func(index int) error {
		if index == 0 {
			return want
		}
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("runSetupWorkers() error = %v, want %v", err, want)
	}
}
