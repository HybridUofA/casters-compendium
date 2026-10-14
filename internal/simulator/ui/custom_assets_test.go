package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestApplyTileCosmeticsUsesPreloadedCardBack(t *testing.T) {
	t.Cleanup(clearCustomAssetImageCache)

	backURL := "https://custom-assets.casterscompendium.com/assets/tts-card-back/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef.png"
	custom := image.NewRGBA(image.Rect(0, 0, 8, 12))
	custom.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	preloadCustomAssetImage(backURL, custom)

	tile := NewCardTile(
		simulatorview.CardView{Face: model.CardFaceDown, ShowFace: false},
		cards.Card{},
		fyne.NewSize(60, 90),
		nil,
		nil,
	)
	applyTileCosmetics(tile, simulatorview.PlayerCosmetics{CardBackURL: backURL})
	if tile.cardBackURL != backURL {
		t.Fatalf("cardBackURL = %q; want %q", tile.cardBackURL, backURL)
	}
	if tile.uprightImage == nil || tile.uprightImage.Image == nil {
		t.Fatal("custom card back image was not applied")
	}
	if tile.uprightImage.Image != custom {
		t.Fatal("tile did not use the preloaded custom card-back image")
	}
}

func TestApplyTileCosmeticsUsesPreloadedCasterToken(t *testing.T) {
	t.Cleanup(clearCustomAssetImageCache)

	tokenURL := "https://custom-assets.casterscompendium.com/assets/simulator-caster-token/fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210.png"
	custom := image.NewRGBA(image.Rect(0, 0, 10, 10))
	custom.Set(1, 1, color.RGBA{R: 9, G: 8, B: 7, A: 255})
	preloadCustomAssetImage(tokenURL, custom)

	tile := NewCardTile(
		simulatorview.CardView{
			CardID:   model.CasterTokenCardID,
			ShowFace: true,
			Face:     model.CardFaceUp,
		},
		cards.Card{ID: string(model.CasterTokenCardID), Name: "Caster Token"},
		fyne.NewSize(60, 90),
		nil,
		nil,
	)
	applyTileCosmetics(tile, simulatorview.PlayerCosmetics{CasterTokenURL: tokenURL})
	if tile.uprightImage == nil || tile.uprightImage.Image != custom {
		t.Fatal("tile did not use the preloaded caster-token image")
	}
}

func TestBoardAppliesSeatCosmeticsToFaceDownTiles(t *testing.T) {
	t.Cleanup(clearCustomAssetImageCache)

	backURL := "https://custom-assets.casterscompendium.com/assets/tts-card-back/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.png"
	custom := image.NewRGBA(image.Rect(0, 0, 4, 6))
	preloadCustomAssetImage(backURL, custom)

	match := testMatchView()
	match.Cosmetics = map[model.PlayerID]simulatorview.PlayerCosmetics{
		"player-one": {CardBackURL: backURL},
	}
	controller := NewBoardController(match, testDefinitions(), BoardActions{}, nil)

	foundCustom := false
	for _, tile := range collectCardTiles(controller.Content()) {
		if tile == nil || tile.View.ShowFace {
			continue
		}
		if tile.cardBackURL == backURL && tile.uprightImage != nil && tile.uprightImage.Image == custom {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Fatal("board did not apply seat cosmetics to a face-down tile")
	}
}

func TestFetchCustomAssetImageRejectsNonAllowlistedURL(t *testing.T) {
	if _, err := fetchCustomAssetImage("https://evil.example/back.png"); err == nil {
		t.Fatal("expected allowlist rejection")
	}
}
