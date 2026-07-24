package deckexport

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/carddata/distribution"
	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type hostedCatalog interface {
	decks.CardCatalog
	All() []gamecards.Card
}

// GenerateHostedTTSAssets builds canonical, reusable face sheets for the whole
// catalog. Placement is sorted by stable card ID, so rebuilding identical input
// produces identical manifests and sheets.
func GenerateHostedTTSAssets(
	outputDirectory string,
	publicDirectoryURL string,
	cardBackURL string,
	catalogVersion string,
	repository hostedCatalog,
	cardImageDirectory string,
) (distribution.TTSManifest, error) {
	if repository == nil {
		return distribution.TTSManifest{}, fmt.Errorf("repository cannot be nil")
	}
	if strings.TrimSpace(outputDirectory) == "" {
		return distribution.TTSManifest{}, fmt.Errorf("output directory cannot be empty")
	}
	if strings.TrimSpace(cardImageDirectory) == "" {
		return distribution.TTSManifest{}, fmt.Errorf("card image directory cannot be empty")
	}
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return distribution.TTSManifest{}, fmt.Errorf("create hosted TTS directory: %w", err)
	}

	cardList := repository.All()
	cardIDs, err := distribution.SortedCardIDs(cardList)
	if err != nil {
		return distribution.TTSManifest{}, err
	}
	manifest := distribution.TTSManifest{
		SchemaVersion:  distribution.SchemaVersion,
		CatalogVersion: strings.TrimSpace(catalogVersion),
		CardBackURL:    strings.TrimSpace(cardBackURL),
		Cards:          make(map[string]distribution.TTSCardLocation, len(cardIDs)),
	}
	publicDirectoryURL = strings.TrimRight(strings.TrimSpace(publicDirectoryURL), "/")

	for start, deckKey := 0, 1; start < len(cardIDs); start, deckKey = start+ttsSheetMaxCards, deckKey+1 {
		end := min(start+ttsSheetMaxCards, len(cardIDs))
		sheetIDs := cardIDs[start:end]
		width, height, err := sheetDimensions(len(sheetIDs))
		if err != nil {
			return distribution.TTSManifest{}, fmt.Errorf("calculate hosted sheet %d dimensions: %w", deckKey, err)
		}
		filename := fmt.Sprintf("sheet-%03d.png", deckKey)
		destination := filepath.Join(outputDirectory, filename)
		if err := writeTTSFileAtomically(destination, func(writer io.Writer) error {
			return writeTTSFaceSheet(writer, sheetIDs, cardImageDirectory)
		}); err != nil {
			return distribution.TTSManifest{}, fmt.Errorf("write hosted sheet %d: %w", deckKey, err)
		}

		manifest.Sheets = append(manifest.Sheets, distribution.TTSSheet{
			DeckKey:   deckKey,
			FaceURL:   publicDirectoryURL + "/" + filename,
			NumWidth:  width,
			NumHeight: height,
			CardCount: len(sheetIDs),
		})
		for slot, cardID := range sheetIDs {
			manifest.Cards[cardID] = distribution.TTSCardLocation{
				DeckKey: deckKey,
				Slot:    slot,
			}
		}
	}
	if err := manifest.Validate(); err != nil {
		return distribution.TTSManifest{}, fmt.Errorf("validate hosted TTS manifest: %w", err)
	}
	return manifest, nil
}

// WriteTTSManifest writes a stable, readable public manifest.
func WriteTTSManifest(writer io.Writer, manifest distribution.TTSManifest) error {
	if writer == nil {
		return fmt.Errorf("TTS manifest writer cannot be nil")
	}
	if err := manifest.Validate(); err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode hosted TTS manifest: %w", err)
	}
	return nil
}

// BuildHostedSavedObject maps a deck onto canonical public sheets. Unlike the
// local exporter, every physical card can refer to a different sheet/deck key.
func BuildHostedSavedObject(
	deck *decks.Deck,
	manifest distribution.TTSManifest,
	repository decks.CardCatalog,
) (SavedObject, error) {
	if deck == nil {
		return SavedObject{}, fmt.Errorf("deck cannot be nil")
	}
	if repository == nil {
		return SavedObject{}, fmt.Errorf("repository cannot be nil")
	}
	if err := manifest.Validate(); err != nil {
		return SavedObject{}, fmt.Errorf("validate hosted TTS manifest: %w", err)
	}

	mainIDs := cardIDsForImageExport(deck.MainDeck, deck.MainOrder)
	if len(mainIDs) == 0 {
		return SavedObject{}, fmt.Errorf("main deck cannot be empty")
	}
	mainTransform := Transform{
		PosX: -2.5, PosY: 1, RotZ: 180,
		ScaleX: 1, ScaleY: 1, ScaleZ: 1,
	}
	mainObject, err := buildHostedDeckObject(
		deck.Name+" - Main Deck", mainIDs, mainTransform, manifest, repository,
	)
	if err != nil {
		return SavedObject{}, fmt.Errorf("build hosted main deck: %w", err)
	}

	objects := []DeckObject{mainObject}
	sideIDs := cardIDsForImageExport(deck.SideDeck, deck.SideOrder)
	if len(sideIDs) > 0 {
		sideTransform := Transform{
			PosX: 2.5, PosY: 1, RotZ: 180,
			ScaleX: 1, ScaleY: 1, ScaleZ: 1,
		}
		sideObject, err := buildHostedDeckObject(
			deck.Name+" - Sideboard", sideIDs, sideTransform, manifest, repository,
		)
		if err != nil {
			return SavedObject{}, fmt.Errorf("build hosted sideboard: %w", err)
		}
		objects = append(objects, sideObject)
	}
	return SavedObject{SaveName: deck.Name, ObjectStates: objects}, nil
}

func buildHostedDeckObject(
	nickname string,
	orderedCardIDs []string,
	transform Transform,
	manifest distribution.TTSManifest,
	repository decks.CardCatalog,
) (DeckObject, error) {
	if strings.TrimSpace(nickname) == "" {
		return DeckObject{}, fmt.Errorf("nickname cannot be empty")
	}

	sheets := make(map[int]distribution.TTSSheet, len(manifest.Sheets))
	for _, sheet := range manifest.Sheets {
		sheets[sheet.DeckKey] = sheet
	}
	customDeck := make(map[string]CustomDeckState)
	deckIDs := make([]int, 0, len(orderedCardIDs))
	cardObjects := make([]CardObject, 0, len(orderedCardIDs))

	for index, cardID := range orderedCardIDs {
		location, found := manifest.Cards[cardID]
		if !found {
			return DeckObject{}, fmt.Errorf("card %q at position %d is absent from hosted catalog", cardID, index+1)
		}
		card, found := repository.FindByID(cardID)
		if !found {
			return DeckObject{}, fmt.Errorf("card %q at position %d was not found", cardID, index+1)
		}
		sheet := sheets[location.DeckKey]
		state := CustomDeckState{
			FaceURL:      sheet.FaceURL,
			BackURL:      manifest.CardBackURL,
			NumWidth:     sheet.NumWidth,
			NumHeight:    sheet.NumHeight,
			BackIsHidden: true,
			UniqueBack:   false,
			Type:         0,
		}
		key := strconv.Itoa(location.DeckKey)
		customDeck[key] = state
		ttsCardID := location.DeckKey*100 + location.Slot
		deckIDs = append(deckIDs, ttsCardID)
		cardObject, err := buildCardObject(ttsCardID, card.Name, location.DeckKey, state)
		if err != nil {
			return DeckObject{}, fmt.Errorf("build card %q at position %d: %w", cardID, index+1, err)
		}
		cardObjects = append(cardObjects, cardObject)
	}

	return DeckObject{
		Name:             "Deck",
		Nickname:         nickname,
		Transform:        transform,
		DeckIDs:          deckIDs,
		CustomDeck:       customDeck,
		ContainedObjects: cardObjects,
	}, nil
}

// InstallHostedTTSDeck writes only the saved-object JSON. Card images remain on
// the immutable public catalog, making the object portable to multiplayer.
func InstallHostedTTSDeck(
	root string,
	deck *decks.Deck,
	manifest distribution.TTSManifest,
	repository decks.CardCatalog,
) (TTSInstallPaths, error) {
	object, err := BuildHostedSavedObject(deck, manifest, repository)
	if err != nil {
		return TTSInstallPaths{}, err
	}
	paths, err := pathPlanner(root, deck.Name, deck.SideTotal() > 0)
	if err != nil {
		return TTSInstallPaths{}, fmt.Errorf("plan hosted TTS installation: %w", err)
	}
	if !isTTSRoot(paths.Root) {
		return TTSInstallPaths{}, fmt.Errorf("TTS root is not usable: %q", paths.Root)
	}
	if err := os.MkdirAll(paths.SavedObjectDirectory, 0o755); err != nil {
		return TTSInstallPaths{}, fmt.Errorf("create TTS saved-object directory: %w", err)
	}
	if err := writeTTSFileAtomically(paths.JSONPath, func(writer io.Writer) error {
		return writeSavedObjectJSON(writer, object)
	}); err != nil {
		return TTSInstallPaths{}, fmt.Errorf("install hosted TTS object: %w", err)
	}

	// Clear unused local-image paths so callers do not imply files were written.
	paths.ImageDirectory = ""
	paths.MainFacePath = ""
	paths.SideFacePath = ""
	paths.BackPath = ""
	return paths, nil
}
