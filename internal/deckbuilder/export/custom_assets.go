package deckexport

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const CustomAssetUploadURL = "https://custom-assets.casterscompendium.com/"

const customAssetHost = "custom-assets.casterscompendium.com"

var (
	customTTSCardBackPath = regexp.MustCompile(
		`^/assets/tts-card-back/[0-9a-f]{64}\.png$`,
	)
	customSimulatorCasterTokenPath = regexp.MustCompile(
		`^/assets/simulator-caster-token/[0-9a-f]{64}\.png$`,
	)
)

// ValidateCustomTTSCardBackURL accepts only immutable images produced by the
// project upload service. Restricting the host prevents deck files from turning
// TTS exports into requests to arbitrary tracking or private-network URLs.
func ValidateCustomTTSCardBackURL(rawURL string) error {
	return validateCustomAssetURL(rawURL, customTTSCardBackPath, "card-back")
}

// ValidateSimulatorCasterTokenURL accepts only immutable caster-token images
// produced by the project upload service.
func ValidateSimulatorCasterTokenURL(rawURL string) error {
	return validateCustomAssetURL(rawURL, customSimulatorCasterTokenPath, "caster-token")
}

// ValidateSimulatorCardBackURL accepts the same immutable card-back URLs used
// for TTS exports; one upload serves both Tabletop Simulator and the desktop
// simulator face-down art.
func ValidateSimulatorCardBackURL(rawURL string) error {
	return ValidateCustomTTSCardBackURL(rawURL)
}

func validateCustomAssetURL(rawURL string, pathPattern *regexp.Regexp, label string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse custom %s URL: %w", label, err)
	}
	if parsed.Scheme != "https" || parsed.Hostname() != customAssetHost || parsed.Port() != "" {
		return fmt.Errorf("custom %s must use the Caster's Compendium upload service", label)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !pathPattern.MatchString(parsed.EscapedPath()) {
		return fmt.Errorf("custom %s URL is not a recognized immutable asset URL", label)
	}
	return nil
}
