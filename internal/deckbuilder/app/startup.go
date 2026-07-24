package deckbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	localdata "github.com/HybridUofA/casters-compendium/internal/carddata/local"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

const (
	applicationID      = localdata.SharedApplicationID
	applicationName    = "Caster's Compendium"
	applicationVersion = "0.1.6"
)

const setupDownloadWorkers = 6

const (
	githubCardDataRootURL = "https://raw.githubusercontent.com/HybridUofA/casters-compendium/main/data"
	githubCardDatabaseURL = githubCardDataRootURL + "/cards.json"
	maxCardDatabaseBytes  = 10 << 20
	cardDatabaseUserAgent = "CastersCompendium/" + applicationVersion
)

type applicationPaths = localdata.Paths

// newApplicationPaths delegates path construction to the shared local-data package.
var newApplicationPaths = localdata.NewPaths

type setupProgress func(message string, current int, total int)
type cardImageURL func(cards.Card) string
type cardSnapshot struct {
	imageURL      cardImageURL
	installedHash string
}

// Run creates the desktop application, prepares local data, and enters the Fyne event loop.
func Run() {
	guiApp := app.NewWithID(applicationID)
	loadAppearanceTheme(guiApp)
	window := guiApp.NewWindow(applicationName)
	window.Resize(fyne.NewSize(1400, 850))

	dataRoot, err := localdata.SharedRoot()
	if err != nil {
		showStartupFailure(window, fmt.Errorf("locate shared application data: %w", err), nil)
		window.ShowAndRun()
		return
	}
	paths := newApplicationPaths(dataRoot)
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
		setWindowContent(window, container.NewCenter(container.NewVBox(
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
	setWindowContent(window, container.NewCenter(container.NewVBox(message, buttons)))
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

	repository, snapshot, err := loadOrDownloadCardDatabase(
		ctx,
		paths.CardDatabase,
		progress,
	)
	if err != nil {
		return nil, err
	}

	cacheImages := cacheCardImages
	if snapshot != nil && snapshot.imageURL != nil {
		cacheImages = func(
			ctx context.Context,
			paths applicationPaths,
			repository *cards.Repository,
			progress setupProgress,
		) error {
			return cacheCardImagesUsing(ctx, paths, repository, snapshot.imageURL, progress)
		}
	}
	if err := cacheImages(ctx, paths, repository, progress); err != nil {
		return nil, err
	}

	installedHash := ""
	if snapshot != nil {
		installedHash = snapshot.installedHash
	}
	if installedHash == "" {
		installedHash, err = hashRepositoryDatabase(repository)
		if err != nil {
			return nil, err
		}
	}
	if err := writeCardListHash(paths.CardListHash, installedHash); err != nil {
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
	return cacheCardImagesUsing(ctx, paths, repository, nil, progress)
}

// cacheGitHubCardImages downloads the version-controlled image snapshot.
func cacheGitHubCardImages(
	ctx context.Context,
	paths applicationPaths,
	repository *cards.Repository,
	progress setupProgress,
) error {
	return cacheCardImagesUsing(ctx, paths, repository, githubCardImageURL, progress)
}

// cacheCardImagesUsing downloads missing images from card or override URLs and builds thumbnails.
func cacheCardImagesUsing(
	ctx context.Context,
	paths applicationPaths,
	repository *cards.Repository,
	imageURL func(cards.Card) string,
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
		var downloadErr error
		if imageURL == nil {
			_, _, downloadErr = cardimages.Download(ctx, httpClient, paths.Images, card)
		} else {
			_, _, downloadErr = cardimages.DownloadFromURL(
				ctx,
				httpClient,
				paths.Images,
				card,
				imageURL(card),
			)
		}
		if downloadErr != nil {
			return fmt.Errorf("download image for %q: %w", card.Name, downloadErr)
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

// githubCardImageURL returns the raw GitHub URL for one version-controlled card image.
func githubCardImageURL(card cards.Card) string {
	return githubCardDataRootURL + "/images/" +
		url.PathEscape(strings.TrimSpace(card.ID)) + ".png"
}

// downloadGitHubCardDatabase retrieves and validates a normalized GitHub snapshot.
func downloadGitHubCardDatabase(
	ctx context.Context,
	client *http.Client,
	databaseURL string,
	progress setupProgress,
) (*cards.Repository, error) {
	if client == nil {
		return nil, fmt.Errorf("download GitHub card database: HTTP client cannot be nil")
	}
	if progress != nil {
		progress("Downloading the GitHub card database snapshot…", 0, 1)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimSpace(databaseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create GitHub card database request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", cardDatabaseUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download GitHub card database: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"download GitHub card database: server returned %s",
			response.Status,
		)
	}

	encoded, err := io.ReadAll(io.LimitReader(response.Body, maxCardDatabaseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read GitHub card database: %w", err)
	}
	if len(encoded) > maxCardDatabaseBytes {
		return nil, fmt.Errorf("GitHub card database exceeds %d MiB", maxCardDatabaseBytes>>20)
	}

	var cardList []cards.Card
	if err := json.Unmarshal(encoded, &cardList); err != nil {
		return nil, fmt.Errorf("decode GitHub card database: %w", err)
	}
	repository, err := cards.NewRepository(cardList)
	if err != nil {
		return nil, fmt.Errorf("validate GitHub card database: %w", err)
	}
	return repository, nil
}

// loadOrDownloadCardDatabase uses local data or installs the fastest current remote source.
func loadOrDownloadCardDatabase(
	ctx context.Context,
	path string,
	progress setupProgress,
) (*cards.Repository, *cardSnapshot, error) {
	if repository, err := cards.LoadFile(path); err == nil {
		return repository, nil, nil
	}

	// The project-hosted snapshot is authoritative for released clients. It is
	// validated before installation and avoids one upstream request per user.
	if client, err := hostedCatalogClientFactory(); err == nil {
		if _, release, err := client.FetchCurrent(ctx); err == nil {
			if repository, encoded, err := client.FetchDatabase(ctx, release); err == nil {
				if err := writeCardDatabaseBytes(path, encoded); err != nil {
					return nil, nil, err
				}
				releaseHash, err := distribution.ReleaseDigest(release)
				if err != nil {
					return nil, nil, err
				}
				imageURL := func(card cards.Card) string {
					result, _ := distribution.CardImageURL(release, card.ID)
					return result
				}
				return repository, &cardSnapshot{
					imageURL:      imageURL,
					installedHash: releaseHash,
				}, nil
			} else {
				log.Printf("hosted card database skipped: %v", err)
			}
		} else {
			log.Printf("hosted catalog skipped: %v", err)
		}
	}

	remote, err := fetchRemoteCardList(ctx, progress)
	if err != nil {
		return nil, nil, err
	}

	githubRepository, githubErr := downloadGitHubCardDatabase(
		ctx,
		remote.client,
		githubCardDatabaseURL,
		progress,
	)
	if githubErr == nil {
		githubHash, hashErr := hashRepositoryCardList(githubRepository)
		if hashErr == nil && githubHash == remote.hash {
			if err := writeCardDatabase(path, githubRepository); err != nil {
				return nil, nil, err
			}
			return githubRepository, &cardSnapshot{
				imageURL:      githubCardImageURL,
				installedHash: remote.hash,
			}, nil
		}
		if hashErr != nil {
			githubErr = hashErr
		} else {
			githubErr = fmt.Errorf("GitHub card database snapshot is not current")
		}
	}
	if githubErr != nil {
		log.Printf("GitHub card database fast path skipped: %v", githubErr)
		if progress != nil {
			progress("GitHub snapshot is not current; rebuilding card data…", 0, 1)
		}
	}
	repository, err := downloadRemoteCardDatabase(ctx, remote, progress)
	if err != nil {
		return nil, nil, err
	}
	if err := writeCardDatabase(path, repository); err != nil {
		return nil, nil, err
	}
	return repository, nil, nil
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
