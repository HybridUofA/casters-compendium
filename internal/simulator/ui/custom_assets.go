package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	deckexport "github.com/HybridUofA/casters-compendium/internal/deckbuilder/export"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const customAssetFetchTimeout = 8 * time.Second
const customAssetMaxBytes = 8 * 1024 * 1024

var (
	customAssetImageCache sync.Map // string -> image.Image
	customAssetFetchOnce  sync.Map // string -> *sync.Once
	customAssetHTTPClient = &http.Client{Timeout: customAssetFetchTimeout}
)

func cosmeticsForPlayer(match simulatorview.MatchView, playerID model.PlayerID) simulatorview.PlayerCosmetics {
	if playerID == "" || len(match.Cosmetics) == 0 {
		return simulatorview.PlayerCosmetics{}
	}
	return match.Cosmetics[playerID]
}

func cachedCustomAssetImage(rawURL string) (image.Image, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, false
	}
	if cached, ok := customAssetImageCache.Load(rawURL); ok {
		return cached.(image.Image), true
	}
	return nil, false
}

// preloadCustomAssetImage stores a decoded image for tests and offline stubs.
func preloadCustomAssetImage(rawURL string, source image.Image) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || source == nil {
		return
	}
	customAssetImageCache.Store(rawURL, source)
}

func clearCustomAssetImageCache() {
	customAssetImageCache.Range(func(key, _ any) bool {
		customAssetImageCache.Delete(key)
		return true
	})
	customAssetFetchOnce.Range(func(key, _ any) bool {
		customAssetFetchOnce.Delete(key)
		return true
	})
}

func fetchCustomAssetImage(rawURL string) (image.Image, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("custom asset URL is empty")
	}
	if err := deckexport.ValidateSimulatorCardBackURL(rawURL); err != nil {
		if err := deckexport.ValidateSimulatorCasterTokenURL(rawURL); err != nil {
			return nil, fmt.Errorf("custom asset URL is not allowlisted")
		}
	}
	if cached, ok := cachedCustomAssetImage(rawURL); ok {
		return cached, nil
	}

	response, err := customAssetHTTPClient.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("custom asset HTTP %d", response.StatusCode)
	}
	limited := io.LimitReader(response.Body, customAssetMaxBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(payload) > customAssetMaxBytes {
		return nil, fmt.Errorf("custom asset exceeds size limit")
	}
	decoded, _, err := image.Decode(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	actual, _ := customAssetImageCache.LoadOrStore(rawURL, decoded)
	return actual.(image.Image), nil
}

func ensureCustomAssetLoaded(rawURL string, onReady func(image.Image)) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || onReady == nil {
		return
	}
	if cached, ok := cachedCustomAssetImage(rawURL); ok {
		onReady(cached)
		return
	}
	onceAny, _ := customAssetFetchOnce.LoadOrStore(rawURL, &sync.Once{})
	once := onceAny.(*sync.Once)
	once.Do(func() {
		go func() {
			decoded, err := fetchCustomAssetImage(rawURL)
			if err != nil {
				customAssetFetchOnce.Delete(rawURL)
				return
			}
			fyne.Do(func() {
				onReady(decoded)
			})
		}()
	})
}

func sidewaysCardBackImageForURL(rawURL string) *canvas.Image {
	if cached, ok := cachedCustomAssetImage(rawURL); ok {
		return newSimulatorCanvasImage(rotateImageClockwise(cached))
	}
	return sidewaysCardBackImage()
}

func applyTileCosmetics(tile *CardTile, cosmetics simulatorview.PlayerCosmetics) {
	if tile == nil {
		return
	}
	if tile.View.ShowFace {
		if tile.View.CardID != model.CasterTokenCardID {
			return
		}
		tokenURL := strings.TrimSpace(cosmetics.CasterTokenURL)
		if tokenURL == "" {
			return
		}
		if cached, ok := cachedCustomAssetImage(tokenURL); ok {
			tile.replaceUprightImage(newSimulatorCanvasImage(cached))
			return
		}
		ensureCustomAssetLoaded(tokenURL, func(decoded image.Image) {
			if tile.View.CardID != model.CasterTokenCardID {
				return
			}
			tile.replaceUprightImage(newSimulatorCanvasImage(decoded))
		})
		return
	}

	backURL := strings.TrimSpace(cosmetics.CardBackURL)
	if backURL == "" {
		return
	}
	if cached, ok := cachedCustomAssetImage(backURL); ok {
		tile.replaceUprightImage(newSimulatorCanvasImage(cached))
		tile.cardBackURL = backURL
		return
	}
	tile.cardBackURL = backURL
	ensureCustomAssetLoaded(backURL, func(decoded image.Image) {
		if tile.View.ShowFace {
			return
		}
		tile.replaceUprightImage(newSimulatorCanvasImage(decoded))
	})
}

func (tile *CardTile) replaceUprightImage(image *canvas.Image) {
	if tile == nil || image == nil {
		return
	}
	wasSideways := tile.sideways
	tile.uprightImage = image
	tile.sidewaysImage = nil
	tile.reversedImage = nil
	tile.setSidewaysState(wasSideways)
	tile.refreshStatusBadge()
	tile.Refresh()
}
