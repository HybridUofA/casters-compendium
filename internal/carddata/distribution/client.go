package distribution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

const (
	maxPointerBytes  = 64 << 10
	maxManifestBytes = 2 << 20
	maxDatabaseBytes = 25 << 20
)

// Client reads public catalog releases. It owns no R2 credentials: every
// desktop request is an ordinary, cacheable HTTPS GET.
type Client struct {
	HTTP       *http.Client
	PointerURL string
	UserAgent  string
}

func (client Client) FetchCurrent(
	ctx context.Context,
) (CatalogPointer, ReleaseManifest, error) {
	var pointer CatalogPointer
	if err := client.fetchJSON(ctx, client.PointerURL, maxPointerBytes, &pointer); err != nil {
		return CatalogPointer{}, ReleaseManifest{}, fmt.Errorf("download catalog pointer: %w", err)
	}
	if err := pointer.Validate(); err != nil {
		return CatalogPointer{}, ReleaseManifest{}, fmt.Errorf("validate catalog pointer: %w", err)
	}

	var release ReleaseManifest
	if err := client.fetchJSON(ctx, pointer.ReleaseURL, maxManifestBytes, &release); err != nil {
		return CatalogPointer{}, ReleaseManifest{}, fmt.Errorf("download catalog release: %w", err)
	}
	if err := release.Validate(); err != nil {
		return CatalogPointer{}, ReleaseManifest{}, fmt.Errorf("validate catalog release: %w", err)
	}
	if release.CatalogVersion != pointer.CatalogVersion {
		return CatalogPointer{}, ReleaseManifest{}, fmt.Errorf(
			"catalog pointer version %q does not match release %q",
			pointer.CatalogVersion,
			release.CatalogVersion,
		)
	}
	return pointer, release, nil
}

func (client Client) FetchDatabase(
	ctx context.Context,
	release ReleaseManifest,
) (*catalog.Repository, []byte, error) {
	if err := release.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validate catalog release: %w", err)
	}
	data, err := client.fetch(ctx, release.Database.URL, maxDatabaseBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("download card database: %w", err)
	}
	if int64(len(data)) != release.Database.Size {
		return nil, nil, fmt.Errorf(
			"card database size is %d bytes; expected %d",
			len(data),
			release.Database.Size,
		)
	}
	if digest := SHA256(data); digest != strings.ToLower(release.Database.SHA256) {
		return nil, nil, fmt.Errorf("card database SHA-256 does not match release manifest")
	}
	var cardList []catalog.Card
	if err := json.Unmarshal(data, &cardList); err != nil {
		return nil, nil, fmt.Errorf("decode card database: %w", err)
	}
	repository, err := catalog.NewRepository(cardList)
	if err != nil {
		return nil, nil, fmt.Errorf("validate card database: %w", err)
	}
	return repository, data, nil
}

func (client Client) FetchTTSManifest(
	ctx context.Context,
	release ReleaseManifest,
) (TTSManifest, error) {
	if err := release.Validate(); err != nil {
		return TTSManifest{}, fmt.Errorf("validate catalog release: %w", err)
	}
	var manifest TTSManifest
	if err := client.fetchJSON(
		ctx,
		release.TabletopSimulator.ManifestURL,
		maxManifestBytes,
		&manifest,
	); err != nil {
		return TTSManifest{}, fmt.Errorf("download TTS manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return TTSManifest{}, fmt.Errorf("validate TTS manifest: %w", err)
	}
	if manifest.CatalogVersion != release.CatalogVersion {
		return TTSManifest{}, fmt.Errorf("TTS manifest catalog version does not match release")
	}
	if manifest.CardBackURL != release.TabletopSimulator.CardBackURL {
		return TTSManifest{}, fmt.Errorf("TTS manifest card back does not match release")
	}
	return manifest, nil
}

func CardImageURL(release ReleaseManifest, cardID string) (string, error) {
	if err := release.Validate(); err != nil {
		return "", err
	}
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return "", fmt.Errorf("card ID cannot be empty")
	}
	return release.Images.BaseURL + url.PathEscape(cardID) + ".png", nil
}

func (client Client) fetchJSON(
	ctx context.Context,
	rawURL string,
	limit int64,
	destination any,
) error {
	data, err := client.fetch(ctx, rawURL, limit)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

func (client Client) fetch(
	ctx context.Context,
	rawURL string,
	limit int64,
) ([]byte, error) {
	if client.HTTP == nil {
		return nil, fmt.Errorf("HTTP client cannot be nil")
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("URL must be absolute HTTPS")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if strings.TrimSpace(client.UserAgent) != "" {
		request.Header.Set("User-Agent", client.UserAgent)
	}
	response, err := client.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("server returned %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}
