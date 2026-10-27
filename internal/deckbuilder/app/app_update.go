package deckbuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	releasenotes "github.com/HybridUofA/casters-compendium/docs/releases"
	"github.com/HybridUofA/casters-compendium/internal/sources/speedrobo"
)

const (
	githubLatestReleaseURL               = "https://api.github.com/repos/HybridUofA/casters-compendium/releases/latest"
	applicationDownloadsURL              = "https://casterscompendium.com/downloads.html"
	appUpdateSkippedVersionPreferenceKey = "app.update.skipped_version"
	maxLatestReleaseBytes                = 1 << 20
)

// latestReleaseFetcher retrieves the newest published application version tag.
// Tests replace this to avoid network access.
var latestReleaseFetcher = fetchLatestReleaseTag

type githubLatestReleaseResponse struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// normalizeApplicationVersion strips a leading "v" for preference storage and display.
func normalizeApplicationVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

// isApplicationOutdated reports whether latest is a newer release than current.
func isApplicationOutdated(currentVersion, latestVersion string) bool {
	current := normalizeApplicationVersion(currentVersion)
	latest := normalizeApplicationVersion(latestVersion)
	if current == "" || latest == "" {
		return false
	}
	return releasenotes.CompareVersions(latest, current) > 0
}

// shouldPromptForApplicationUpdate decides whether to show the outdated dialog.
// Manual checks always prompt when outdated; automatic checks honor a skipped version.
func shouldPromptForApplicationUpdate(
	currentVersion string,
	latestVersion string,
	skippedVersion string,
	manual bool,
) bool {
	if !isApplicationOutdated(currentVersion, latestVersion) {
		return false
	}
	if manual {
		return true
	}
	return normalizeApplicationVersion(skippedVersion) != normalizeApplicationVersion(latestVersion)
}

// fetchLatestReleaseTag reads the current stable GitHub release tag.
func fetchLatestReleaseTag(ctx context.Context) (string, error) {
	client, err := speedrobo.NewClient()
	if err != nil {
		return "", err
	}
	// Startup checks should fail quickly when offline rather than block the UI.
	client.Timeout = 8 * time.Second

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, githubLatestReleaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("create latest-release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", cardDatabaseUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("fetch latest release: server returned %s", response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxLatestReleaseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read latest release: %w", err)
	}
	if len(body) > maxLatestReleaseBytes {
		return "", fmt.Errorf("latest release response exceeds %d bytes", maxLatestReleaseBytes)
	}

	var payload githubLatestReleaseResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return "", fmt.Errorf("latest release is missing tag_name")
	}
	if payload.Draft || payload.Prerelease {
		return "", fmt.Errorf("latest release %q is not a stable publication", tag)
	}
	return tag, nil
}

// scheduleApplicationUpdateCheck quietly compares the installed version to GitHub.
// Failures stay silent so offline launches are unaffected.
func scheduleApplicationUpdateCheck(window fyne.Window) {
	go runApplicationUpdateCheck(window, false)
}

// checkForApplicationUpdate runs an explicit menu-driven update check.
func checkForApplicationUpdate(window fyne.Window) {
	go runApplicationUpdateCheck(window, true)
}

func runApplicationUpdateCheck(window fyne.Window, manual bool) {
	latest, err := latestReleaseFetcher(context.Background())
	if err != nil {
		log.Printf("application update check skipped: %v", err)
		if manual {
			fyne.Do(func() {
				dialog.ShowInformation(
					"Update Check Failed",
					"Could not reach the release server. Check your connection and try again.",
					window,
				)
			})
		}
		return
	}

	skipped := ""
	if guiApp := fyne.CurrentApp(); guiApp != nil {
		skipped = guiApp.Preferences().StringWithFallback(
			appUpdateSkippedVersionPreferenceKey,
			"",
		)
	}

	if !shouldPromptForApplicationUpdate(applicationVersion, latest, skipped, manual) {
		if manual {
			fyne.Do(func() {
				dialog.ShowInformation(
					"Up to Date",
					fmt.Sprintf(
						"You are running the latest release (v%s).",
						normalizeApplicationVersion(applicationVersion),
					),
					window,
				)
			})
		}
		return
	}

	fyne.Do(func() {
		showApplicationUpdateAvailableDialog(window, applicationVersion, latest)
	})
}

// showApplicationUpdateAvailableDialog offers downloads, postpone, or skip-this-version.
func showApplicationUpdateAvailableDialog(
	window fyne.Window,
	currentVersion string,
	latestVersion string,
) {
	message := widget.NewLabel(fmt.Sprintf(
		"A newer version is available.\n\nYou are running v%s.\nLatest release is v%s.\n\nOpen the downloads page to update?",
		normalizeApplicationVersion(currentVersion),
		normalizeApplicationVersion(latestVersion),
	))
	message.Wrapping = fyne.TextWrapWord
	message.Alignment = fyne.TextAlignCenter

	var popup dialog.Dialog
	openButton := widget.NewButton("Open Downloads", func() {
		if popup != nil {
			popup.Hide()
		}
		openApplicationDownloadsPage(window)
	})
	openButton.Importance = widget.HighImportance
	laterButton := widget.NewButton("Later", func() {
		if popup != nil {
			popup.Hide()
		}
	})
	skipButton := widget.NewButton("Skip This Version", func() {
		if guiApp := fyne.CurrentApp(); guiApp != nil {
			guiApp.Preferences().SetString(
				appUpdateSkippedVersionPreferenceKey,
				normalizeApplicationVersion(latestVersion),
			)
		}
		if popup != nil {
			popup.Hide()
		}
	})

	content := container.NewVBox(
		message,
		widget.NewSeparator(),
		container.NewGridWithColumns(3, skipButton, laterButton, openButton),
	)
	popup = dialog.NewCustomWithoutButtons("Application Update Available", content, window)
	popup.Resize(fyne.NewSize(420, 220))
	popup.Show()
}

// openApplicationDownloadsPage opens the public downloads page in the system browser.
func openApplicationDownloadsPage(window fyne.Window) {
	downloadsURL, err := url.Parse(applicationDownloadsURL)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}
	if err := fyne.CurrentApp().OpenURL(downloadsURL); err != nil {
		dialog.ShowError(err, window)
	}
}
