package deckexport

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const CustomAssetUploadURL = "https://custom-assets.casterscompendium.com/"

var customTTSCardBackPath = regexp.MustCompile(
	`^/assets/tts-card-back/[0-9a-f]{64}\.png$`,
)

// ValidateCustomTTSCardBackURL accepts only immutable images produced by the
// project upload service. Restricting the host prevents deck files from turning
// TTS exports into requests to arbitrary tracking or private-network URLs.
func ValidateCustomTTSCardBackURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse custom card-back URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() != "custom-assets.casterscompendium.com" || parsed.Port() != "" {
		return fmt.Errorf("custom card back must use the Caster's Compendium upload service")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !customTTSCardBackPath.MatchString(parsed.EscapedPath()) {
		return fmt.Errorf("custom card-back URL is not a recognized immutable asset URL")
	}
	return nil
}
