package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/speedrobo"
)

const (
	applicationID   = "io.github.hybriduofa.casterdeckbuilder"
	applicationName = "Caster's Compendium"
)

const setupDownloadWorkers = 6

type applicationPaths struct {
	Root          string
	CardDatabase  string
	CardListHash  string
	Images        string
	Thumbnails    string
	SetupComplete string
}

// newApplicationPaths derives every managed data path from Fyne's per-application root.
func newApplicationPaths(root string) applicationPaths {
	return applicationPaths{
		Root:          root,
		CardDatabase:  filepath.Join(root, "cards.json"),
		CardListHash:  filepath.Join(root, "cardlist.sha256"),
		Images:        filepath.Join(root, "images"),
		Thumbnails:    filepath.Join(root, "thumbnails"),
		SetupComplete: filepath.Join(root, ".setup-complete-v1"),
	}
}

type setupProgress func(message string, current int, total int)

// main creates the desktop application, prepares local data, and enters the Fyne event loop.
func main() {
	guiApp := app.NewWithID(applicationID)
	window := guiApp.NewWindow(applicationName)
	window.Resize(fyne.NewSize(1400, 850))

	paths := newApplicationPaths(guiApp.Storage().RootURI().Path())
	cardimages.ConfigureDirectories(paths.Images, paths.Thumbnails)

	if err := os.MkdirAll(paths.Root, 0o755); err != nil {
		showStartupFailure(window, fmt.Errorf("create application data directory: %w", err), nil)
		window.ShowAndRun()
		return
	}
	if err := cardimages.InstallBundledCardBack(paths.Images); err != nil {
		showStartupFailure(window, err, nil)
		window.ShowAndRun()
		return
	}

	if repository, ready := loadReadyApplicationData(paths); ready {
		checkForCardDatabaseUpdate(window, paths, repository)
		window.ShowAndRun()
		return
	}

	var runSetup func()
	runSetup = func() {
		status := widget.NewLabel("Preparing first-time setup…")
		status.Wrapping = fyne.TextWrapWord
		progressBar := widget.NewProgressBar()
		window.SetContent(container.NewCenter(container.NewVBox(
			widget.NewLabelWithStyle(
				applicationName+" Setup",
				fyne.TextAlignCenter,
				fyne.TextStyle{Bold: true},
			),
			status,
			progressBar,
		)))

		go func() {
			repository, err := initializeApplicationData(
				context.Background(),
				paths,
				func(message string, current int, total int) {
					fyne.Do(func() {
						status.SetText(message)
						if total > 0 {
							progressBar.SetValue(float64(current) / float64(total))
						}
					})
				},
			)
			fyne.Do(func() {
				if err != nil {
					showStartupFailure(window, err, runSetup)
					return
				}
				showApplication(window, paths, repository)
			})
		}()
	}

	runSetup()
	window.ShowAndRun()
}

// loadReadyApplicationData returns the repository only when setup and every cache entry are complete.
func loadReadyApplicationData(paths applicationPaths) (*cards.Repository, bool) {
	if _, err := os.Stat(paths.SetupComplete); err != nil {
		return nil, false
	}
	repository, err := cards.LoadFile(paths.CardDatabase)
	if err != nil {
		return nil, false
	}
	for _, card := range repository.All() {
		if _, found := cardimages.FindIn(paths.Images, card.ID); !found {
			return nil, false
		}
		if _, found := cardimages.FindThumbnail(card.ID); !found {
			return nil, false
		}
	}
	return repository, true
}

// showStartupFailure replaces the window content with a recoverable setup error screen.
func showStartupFailure(window fyne.Window, err error, retry func()) {
	message := widget.NewLabel("Setup could not be completed:\n\n" + err.Error())
	message.Wrapping = fyne.TextWrapWord
	buttons := container.NewGridWithColumns(1, widget.NewButton("Quit", func() {
		window.Close()
	}))
	if retry != nil {
		buttons = container.NewGridWithColumns(2,
			widget.NewButton("Retry", retry),
			widget.NewButton("Quit", func() { window.Close() }),
		)
	}
	window.SetContent(container.NewCenter(container.NewVBox(message, buttons)))
}

// initializeApplicationData installs bundled assets and prepares the database and image caches.
func initializeApplicationData(
	ctx context.Context,
	paths applicationPaths,
	progress setupProgress,
) (*cards.Repository, error) {
	if err := os.MkdirAll(paths.Images, 0o755); err != nil {
		return nil, fmt.Errorf("create image directory: %w", err)
	}
	if err := os.MkdirAll(paths.Thumbnails, 0o755); err != nil {
		return nil, fmt.Errorf("create thumbnail directory: %w", err)
	}
	if err := cardimages.InstallBundledCardBack(paths.Images); err != nil {
		return nil, err
	}

	repository, err := loadOrDownloadCardDatabase(ctx, paths.CardDatabase, progress)
	if err != nil {
		return nil, err
	}

	if err := cacheCardImages(ctx, paths, repository, progress); err != nil {
		return nil, err
	}

	cardListHash, err := hashRepositoryCardList(repository)
	if err != nil {
		return nil, err
	}
	if err := writeCardListHash(paths.CardListHash, cardListHash); err != nil {
		return nil, err
	}

	progress("Setup complete", 1, 1)
	if err := os.WriteFile(paths.SetupComplete, []byte("complete\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write setup marker: %w", err)
	}
	return repository, nil
}

// cacheCardImages downloads missing full images and generates their thumbnails concurrently.
func cacheCardImages(
	ctx context.Context,
	paths applicationPaths,
	repository *cards.Repository,
	progress setupProgress,
) error {
	httpClient, err := speedrobo.NewClient()
	if err != nil {
		return err
	}
	cardList := repository.All()
	completedImages := 0
	var imageProgress sync.Mutex
	err = runSetupWorkers(ctx, len(cardList), setupDownloadWorkers, func(index int) error {
		card := cardList[index]
		if _, _, err := cardimages.Download(ctx, httpClient, paths.Images, card); err != nil {
			return fmt.Errorf("download image for %q: %w", card.Name, err)
		}
		if _, found := cardimages.FindThumbnail(card.ID); !found {
			if _, err := cardimages.CreateThumbnail(card.ID); err != nil {
				return fmt.Errorf("create thumbnail for %q: %w", card.Name, err)
			}
		}

		imageProgress.Lock()
		defer imageProgress.Unlock()
		completedImages++
		if progress != nil {
			progress(
				fmt.Sprintf(
					"Downloading card images (%d/%d): %s",
					completedImages,
					len(cardList),
					card.Name,
				),
				completedImages,
				len(cardList),
			)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// loadOrDownloadCardDatabase uses a valid local database or downloads and stores a new one.
func loadOrDownloadCardDatabase(
	ctx context.Context,
	path string,
	progress setupProgress,
) (*cards.Repository, error) {
	if repository, err := cards.LoadFile(path); err == nil {
		return repository, nil
	}

	remote, err := fetchRemoteCardList(ctx, progress)
	if err != nil {
		return nil, err
	}
	repository, err := downloadRemoteCardDatabase(ctx, remote, progress)
	if err != nil {
		return nil, err
	}
	if err := writeCardDatabase(path, repository); err != nil {
		return nil, err
	}
	return repository, nil
}

// runSetupWorkers executes indexed work with bounded concurrency and cancels after the first error.
func runSetupWorkers(
	ctx context.Context,
	itemCount int,
	workerCount int,
	work func(index int) error,
) error {
	if itemCount == 0 {
		return nil
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > itemCount {
		workerCount = itemCount
	}

	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan int)
	errors := make(chan error, 1)

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-workerContext.Done():
					return
				case index, open := <-jobs:
					if !open {
						return
					}
					if err := work(index); err != nil {
						select {
						case errors <- err:
						default:
						}
						cancel()
						return
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index := range itemCount {
			select {
			case <-workerContext.Done():
				return
			case jobs <- index:
			}
		}
	}()

	workers.Wait()
	select {
	case err := <-errors:
		return err
	default:
		return ctx.Err()
	}
}

// writeFileAtomically replaces a file only after its complete contents are safely closed.
func writeFileAtomically(path string, data []byte) (err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".cards-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer os.Remove(temporaryPath)
	defer func() {
		if !closed {
			if closeErr := temporary.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
		}
	}()

	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.Rename(temporaryPath, path); err == nil {
		return nil
	}
	if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(temporaryPath, path)
}
