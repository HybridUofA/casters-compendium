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
	encoded, err := json.MarshalIndent(repository.All(), "", " ")
	if err != nil {
		return fmt.Errorf("encode card database: %w", err)
	}
	encoded = append(encoded, '\n')
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

// checkForCardDatabaseUpdate performs the lightweight startup comparison and prompts on change.
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
		localHash, err := hashRepositoryCardList(repository)
		if err != nil {
			log.Printf("card database update check skipped: %v", err)
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}
		if storedHash, readErr := readCardListHash(paths.CardListHash); readErr != nil || storedHash != localHash {
			if writeErr := writeCardListHash(paths.CardListHash, localHash); writeErr != nil {
				log.Printf("could not record installed card-list hash: %v", writeErr)
			}
		}

		remote, err := fetchRemoteCardList(context.Background(), nil)
		if err != nil {
			log.Printf("card database update check skipped: %v", err)
			fyne.Do(func() { showApplication(window, paths, repository) })
			return
		}
		if remote.hash == localHash {
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
					runCardDatabaseUpdate(window, paths, repository, remote)
				},
				window,
			)
		})
	}()
}

// confirmManualCardDatabaseUpdate asks before starting a user-requested full refresh.
func confirmManualCardDatabaseUpdate(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {
	dialog.ShowConfirm(
		"Update Card Database",
		"Download and rebuild the latest card database? Existing deck files will not be changed.",
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
	remote *remoteCardList,
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
		updatedRepository, err := updateApplicationData(
			context.Background(),
			paths,
			currentRepository,
			remote,
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
