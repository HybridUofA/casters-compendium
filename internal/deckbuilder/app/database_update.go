package deckbuilder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	cardupdate "github.com/HybridUofA/casters-compendium/internal/carddata/update"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

type remoteCardList struct {
	client    *http.Client
	config    speedrobo.PageConfig
	summaries []speedrobo.CardResponse
	hash      string
}

type cardListHashEntry struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ImageURL      string `json:"image_url"`
	Expansion     string `json:"expansion"`
	IsPlaytesting bool   `json:"is_playtesting"`
}

// hashRepositoryCardList hashes the stable summary fields of the installed normalized cards.
func hashRepositoryCardList(repository *cards.Repository) (string, error) {
	if repository == nil {
		return "", fmt.Errorf("card repository cannot be nil")
	}

	cardList := repository.All()
	entries := make([]cardListHashEntry, len(cardList))
	for index, card := range cardList {
		entries[index] = cardListHashEntry{
			ID:            strings.TrimSpace(card.ID),
			Name:          strings.ToLower(strings.TrimSpace(card.Name)),
			ImageURL:      strings.TrimSpace(card.ImageURL),
			Expansion:     strings.TrimSpace(card.Expansion),
			IsPlaytesting: card.IsPlaytesting,
		}
	}
	return hashCardListEntries(entries)
}

// hashRepositoryDatabase covers the complete normalized record rather than the
// lightweight upstream summary used by legacy update checks.
func hashRepositoryDatabase(repository *cards.Repository) (string, error) {
	if repository == nil {
		return "", fmt.Errorf("card repository cannot be nil")
	}
	encoded, err := distribution.EncodeCards(repository.All())
	if err != nil {
		return "", err
	}
	return distribution.SHA256(encoded), nil
}

// hashRemoteCardList hashes remote summary records using the same canonical representation.
func hashRemoteCardList(summaries []speedrobo.CardResponse) (string, error) {
	entries := make([]cardListHashEntry, len(summaries))
	for index, summary := range summaries {
		isPlaytesting, err := parseRemotePlaytesting(summary.PlayTesting)
		if err != nil {
			return "", fmt.Errorf("card %q: %w", summary.CardKey, err)
		}
		entries[index] = cardListHashEntry{
			ID:            strings.TrimSpace(summary.ID),
			Name:          strings.ToLower(strings.TrimSpace(summary.CardKey)),
			ImageURL:      strings.TrimSpace(summary.ImageURL),
			Expansion:     strings.TrimSpace(summary.Expansion),
			IsPlaytesting: isPlaytesting,
		}
	}
	return hashCardListEntries(entries)
}

// parseRemotePlaytesting normalizes the API's numeric or textual boolean representation.
func parseRemotePlaytesting(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "0", "false":
		return false, nil
	case "1", "true":
		return true, nil
	default:
		return false, fmt.Errorf("unexpected playtesting value %q", value)
	}
}

// hashCardListEntries sorts canonical entries and returns their deterministic SHA-256 digest.
func hashCardListEntries(entries []cardListHashEntry) (string, error) {
	entries = append([]cardListHashEntry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]
		if left.ID != right.ID {
			return left.ID < right.ID
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.ImageURL != right.ImageURL {
			return left.ImageURL < right.ImageURL
		}
		if left.Expansion != right.Expansion {
			return left.Expansion < right.Expansion
		}
		return !left.IsPlaytesting && right.IsPlaytesting
	})

	encoded, err := json.Marshal(entries)
	if err != nil {
		return "", fmt.Errorf("encode card-list hash input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

// readCardListHash loads and validates a stored SHA-256 card-list digest.
func readCardListHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := strings.ToLower(strings.TrimSpace(string(data)))
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("invalid card-list SHA-256 hash")
	}
	return hash, nil
}

// writeCardListHash validates and atomically records a card-list digest.
func writeCardListHash(path string, hash string) error {
	hash = strings.ToLower(strings.TrimSpace(hash))
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("write card-list hash: invalid SHA-256 hash")
	}
	if err := writeFileAtomically(path, []byte(hash+"\n")); err != nil {
		return fmt.Errorf("write card-list hash: %w", err)
	}
	return nil
}

// fetchRemoteCardList downloads card summaries and computes the lightweight update digest.
func fetchRemoteCardList(
	ctx context.Context,
	progress setupProgress,
) (*remoteCardList, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if progress != nil {
		progress("Connecting to the card database…", 0, 1)
	}

	client, err := speedrobo.NewClient()
	if err != nil {
		return nil, err
	}
	config, err := speedrobo.FetchPageConfig(client)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if progress != nil {
		progress("Checking the current card list…", 0, 1)
	}
	summaries, err := speedrobo.FetchAllCards(client, config)
	if err != nil {
		return nil, err
	}
	hash, err := hashRemoteCardList(summaries)
	if err != nil {
		return nil, err
	}
	return &remoteCardList{
		client:    client,
		config:    config,
		summaries: summaries,
		hash:      hash,
	}, nil
}

// downloadRemoteCardDatabase fetches and normalizes every detailed card record concurrently.
func downloadRemoteCardDatabase(
	ctx context.Context,
	remote *remoteCardList,
	progress setupProgress,
) (*cards.Repository, error) {
	if remote == nil {
		return nil, fmt.Errorf("remote card list cannot be nil")
	}

	normalized := make([]cards.Card, len(remote.summaries))
	completedDetails := 0
	var detailProgress sync.Mutex
	err := runSetupWorkers(ctx, len(remote.summaries), setupDownloadWorkers, func(index int) error {
		summary := remote.summaries[index]
		response, err := speedrobo.FetchCardDetails(
			remote.client,
			remote.config.AjaxURL,
			remote.config.Nonce,
			summary.ID,
		)
		if err != nil {
			return fmt.Errorf("download data for %q: %w", summary.CardKey, err)
		}
		card, err := cardupdate.FromSpeedrobo(response.Data.Card)
		if err != nil {
			return fmt.Errorf("normalize %q: %w", summary.CardKey, err)
		}
		normalized[index] = card

		detailProgress.Lock()
		defer detailProgress.Unlock()
		completedDetails++
		if progress != nil {
			progress(
				fmt.Sprintf(
					"Downloading card data (%d/%d): %s",
					completedDetails,
					len(remote.summaries),
					summary.CardKey,
				),
				completedDetails,
				len(remote.summaries),
			)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cards.NewRepository(normalized)
}

// writeCardDatabase atomically stores a repository in the normalized JSON format.
func writeCardDatabase(path string, repository *cards.Repository) error {
	if repository == nil {
		return fmt.Errorf("write card database: repository cannot be nil")
	}
	encoded, err := distribution.EncodeCards(repository.All())
	if err != nil {
		return err
	}
	return writeCardDatabaseBytes(path, encoded)
}

func writeCardDatabaseBytes(path string, encoded []byte) error {
	if len(encoded) == 0 {
		return fmt.Errorf("write card database: encoded database cannot be empty")
	}
	if err := writeFileAtomically(path, encoded); err != nil {
		return fmt.Errorf("write card database: %w", err)
	}
	return nil
}

// updateApplicationData rebuilds card data and caches before publishing the new local database.
func updateApplicationData(
	ctx context.Context,
	paths applicationPaths,
	currentRepository *cards.Repository,
	remote *remoteCardList,
	progress setupProgress,
) (*cards.Repository, error) {
	if remote == nil {
		var err error
		remote, err = fetchRemoteCardList(ctx, progress)
		if err != nil {
			return nil, err
		}
	}

	updatedRepository, err := downloadRemoteCardDatabase(ctx, remote, progress)
	if err != nil {
		return nil, err
	}
	if err := invalidateChangedCardImages(paths, currentRepository, updatedRepository); err != nil {
		return nil, err
	}
	if err := cacheCardImages(ctx, paths, updatedRepository, progress); err != nil {
		return nil, err
	}
	if err := writeCardDatabase(paths.CardDatabase, updatedRepository); err != nil {
		return nil, err
	}
	updatedHash, err := hashRepositoryCardList(updatedRepository)
	if err != nil {
		return nil, err
	}
	if err := writeCardListHash(paths.CardListHash, updatedHash); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.SetupComplete, []byte("complete\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write setup marker: %w", err)
	}
	return updatedRepository, nil
}

// invalidateChangedCardImages removes cached art whose source URL changed between databases.
func invalidateChangedCardImages(
	paths applicationPaths,
	currentRepository *cards.Repository,
	updatedRepository *cards.Repository,
) error {
	if currentRepository == nil || updatedRepository == nil {
		return nil
	}
	currentCards := make(map[string]cards.Card)
	for _, card := range currentRepository.All() {
		currentCards[card.ID] = card
	}
	for _, updated := range updatedRepository.All() {
		current, found := currentCards[updated.ID]
		if !found || strings.TrimSpace(current.ImageURL) == strings.TrimSpace(updated.ImageURL) {
			continue
		}
		if imagePath, found := cardimages.FindIn(paths.Images, updated.ID); found {
			if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove outdated image for %q: %w", updated.Name, err)
			}
		}
		if thumbnailPath, found := cardimages.FindThumbnail(updated.ID); found {
			if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove outdated thumbnail for %q: %w", updated.Name, err)
			}
		}
	}
	return nil
}

// invalidateHostedCardImages ensures an artwork-only catalog correction is not
// hidden by the local cache. It removes only known card files and leaves the
// bundled card back and unrelated application data untouched.
func invalidateHostedCardImages(
	paths applicationPaths,
	repositories ...*cards.Repository,
) error {
	seen := make(map[string]struct{})
	for _, repository := range repositories {
		if repository == nil {
			continue
		}
		for _, card := range repository.All() {
			if _, duplicate := seen[card.ID]; duplicate {
				continue
			}
			seen[card.ID] = struct{}{}
			if imagePath, found := cardimages.FindIn(paths.Images, card.ID); found {
				if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove old hosted image for %q: %w", card.Name, err)
				}
			}
			if thumbnailPath, found := cardimages.FindThumbnail(card.ID); found {
				if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove old hosted thumbnail for %q: %w", card.Name, err)
				}
			}
		}
	}
	return nil
}

func updateApplicationDataFromHosted(
	ctx context.Context,
	paths applicationPaths,
	currentRepository *cards.Repository,
	release *distribution.ReleaseManifest,
	progress setupProgress,
) (*cards.Repository, error) {
	client, err := hostedCatalogClientFactory()
	if err != nil {
		return nil, err
	}
	if release == nil {
		_, currentRelease, err := client.FetchCurrent(ctx)
		if err != nil {
			return nil, err
		}
		release = &currentRelease
	}
	if progress != nil {
		progress("Downloading the published card database…", 0, 1)
	}
	updatedRepository, encoded, err := client.FetchDatabase(ctx, *release)
	if err != nil {
		return nil, err
	}
	if err := invalidateHostedCardImages(paths, currentRepository, updatedRepository); err != nil {
		return nil, err
	}
	imageURL := func(card cards.Card) string {
		result, _ := distribution.CardImageURL(*release, card.ID)
		return result
	}
	if err := cacheCardImagesUsing(ctx, paths, updatedRepository, imageURL, progress); err != nil {
		return nil, err
	}
	if err := writeCardDatabaseBytes(paths.CardDatabase, encoded); err != nil {
		return nil, err
	}
	releaseHash, err := distribution.ReleaseDigest(*release)
	if err != nil {
		return nil, err
	}
	if err := writeCardListHash(paths.CardListHash, releaseHash); err != nil {
		return nil, err
	}
	if err := os.WriteFile(paths.SetupComplete, []byte("complete\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write setup marker: %w", err)
	}
	return updatedRepository, nil
}

// checkForCardDatabaseUpdate compares the complete installed database digest to
// the maintainer-approved hosted release and prompts only after validation.
func checkForCardDatabaseUpdate(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {
	window.SetTitle(applicationName)
	window.SetContent(databaseProgressContent(
		"Checking for Card Updates",
		"Comparing the installed card list with the latest version…",
		widget.NewProgressBarInfinite(),
	))

	go func() {
		storedHash, _ := readCardListHash(paths.CardListHash)

		client, err := hostedCatalogClientFactory()
		if err != nil {
			log.Printf("card database update check skipped: %v", err)
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}
		_, release, err := client.FetchCurrent(context.Background())
		if err != nil {
			log.Printf("card database update check skipped: %v", err)
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}
		releaseHash, err := distribution.ReleaseDigest(release)
		if err != nil {
			log.Printf("card database update check skipped: %v", err)
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}
		if strings.EqualFold(storedHash, releaseHash) {
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}

		fyne.Do(func() {
			dialog.ShowConfirm(
				"Card Database Update Available",
				"The available card list differs from the installed database. Update now? Existing deck files will not be changed.",
				func(update bool) {
					if !update {
						showApplication(window, paths, repository)
						return
					}
					runCardDatabaseUpdate(window, paths, repository, &release)
				},
				window,
			)
		})
	}()
}

// confirmManualCardDatabaseUpdate asks before installing the latest published snapshot.
func confirmManualCardDatabaseUpdate(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {
	dialog.ShowConfirm(
		"Update Card Database",
		"Download and install the latest published card database? Existing deck files will not be changed.",
		func(update bool) {
			if update {
				runCardDatabaseUpdate(window, paths, repository, nil)
			}
		},
		window,
	)
}

// runCardDatabaseUpdate drives the asynchronous refresh screen and installs the resulting repository.
func runCardDatabaseUpdate(
	window fyne.Window,
	paths applicationPaths,
	currentRepository *cards.Repository,
	release *distribution.ReleaseManifest,
) {
	status := widget.NewLabel("Preparing database update…")
	status.Wrapping = fyne.TextWrapWord
	progressBar := widget.NewProgressBar()
	window.SetTitle(applicationName + " — Updating Card Database")
	window.SetContent(databaseProgressContent(
		"Updating Card Database",
		"",
		container.NewVBox(status, progressBar),
	))

	go func() {
		updatedRepository, err := updateApplicationDataFromHosted(
			context.Background(),
			paths,
			currentRepository,
			release,
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
				showApplication(window, paths, currentRepository)
				dialog.ShowError(err, window)
				return
			}
			showApplication(window, paths, updatedRepository)
			dialog.ShowInformation(
				"Card Database Updated",
				"The latest card data and images are ready.",
				window,
			)
		})
	}()
}

// databaseProgressContent creates a centered status panel for checking or updating card data.
func databaseProgressContent(
	title string,
	message string,
	progress fyne.CanvasObject,
) fyne.CanvasObject {
	objects := []fyne.CanvasObject{
		widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}
	if strings.TrimSpace(message) != "" {
		label := widget.NewLabel(message)
		label.Alignment = fyne.TextAlignCenter
		label.Wrapping = fyne.TextWrapWord
		objects = append(objects, label)
	}
	objects = append(objects, progress)
	return container.NewCenter(container.NewVBox(objects...))
}
