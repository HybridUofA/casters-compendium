package deckexport

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type SavedObject struct {
	SaveName     string       `json:"SaveName"`
	ObjectStates []DeckObject `json:"ObjectStates"`
}

type DeckObject struct {
	Name             string                     `json:"Name"`
	Nickname         string                     `json:"Nickname"`
	Description      string                     `json:"Description"`
	Transform        Transform                  `json:"Transform"`
	DeckIDs          []int                      `json:"DeckIDs"`
	CustomDeck       map[string]CustomDeckState `json:"CustomDeck"`
	ContainedObjects []CardObject               `json:"ContainedObjects"`
}

type CustomDeckState struct {
	FaceURL      string `json:"FaceURL"`
	BackURL      string `json:"BackURL"`
	NumWidth     int    `json:"NumWidth"`
	NumHeight    int    `json:"NumHeight"`
	BackIsHidden bool   `json:"BackIsHidden"`
	UniqueBack   bool   `json:"UniqueBack"`
	Type         int    `json:"Type"`
}

type CardObject struct {
	Name        string                     `json:"Name"`
	Nickname    string                     `json:"Nickname"`
	Description string                     `json:"Description"`
	CardID      int                        `json:"CardID"`
	Transform   Transform                  `json:"Transform"`
	CustomDeck  map[string]CustomDeckState `json:"CustomDeck"`
}

type Transform struct {
	PosX   float64 `json:"posX"`
	PosY   float64 `json:"posY"`
	PosZ   float64 `json:"posZ"`
	RotX   float64 `json:"rotX"`
	RotY   float64 `json:"rotY"`
	RotZ   float64 `json:"rotZ"`
	ScaleX float64 `json:"scaleX"`
	ScaleY float64 `json:"scaleY"`
	ScaleZ float64 `json:"scaleZ"`
}

const ttsSheetColumns int = 10
const ttsSheetMaxRows int = 7
const ttsSheetMaxCards = ttsSheetColumns * ttsSheetMaxRows

func genDeckIDs(
	orderedCardIDs []string,
	deckKey int,
) (physicalDeckIDs []int, uniqueSheetIDs []string, err error) {
	var slotByCardID = make(map[string]int)
	if deckKey <= 0 {
		return nil, nil, fmt.Errorf("deck key must be positive: %d", deckKey)
	}
	for _, cardID := range orderedCardIDs {
		if cardID == "" {
			return nil, nil, fmt.Errorf("card ID cannot be blank")
		}
		slot, found := slotByCardID[cardID]
		if !found {
			if len(uniqueSheetIDs) >= ttsSheetMaxCards {
				return nil, nil, fmt.Errorf("more cards than sheet allows: %d, max %d", len(uniqueSheetIDs)+1, ttsSheetMaxCards)
			}
			slot = len(uniqueSheetIDs)
			slotByCardID[cardID] = slot
			uniqueSheetIDs = append(uniqueSheetIDs, cardID)
		}

		ttsID := deckKey*100 + slot
		physicalDeckIDs = append(physicalDeckIDs, ttsID)
	}
	return physicalDeckIDs, uniqueSheetIDs, nil
}

func sheetDimensions(uniqueCardCount int) (int, int, error) {
	var width int
	var height int
	if uniqueCardCount <= 0 {
		return 0, 0, fmt.Errorf("card count must be positive: %d", uniqueCardCount)
	}
	if uniqueCardCount > ttsSheetMaxCards {
		return 0, 0, fmt.Errorf("card count must be at most %d max cards: %d", ttsSheetMaxCards, uniqueCardCount)
	}
	if uniqueCardCount < ttsSheetColumns {
		width = uniqueCardCount
	} else {
		width = ttsSheetColumns
	}
	height = (uniqueCardCount + width - 1) / width
	return width, height, nil
}

func buildCustomDeckState(faceURL string, backURL string, uniqueCardCount int) (CustomDeckState, error) {
	if faceURL == "" {
		return CustomDeckState{}, fmt.Errorf("face path must be specified")
	}
	if backURL == "" {
		return CustomDeckState{}, fmt.Errorf("back path must be specified")
	}
	width, height, err := sheetDimensions(uniqueCardCount)
	if err != nil {
		return CustomDeckState{}, err
	}
	state := CustomDeckState{
		FaceURL:      faceURL,
		BackURL:      backURL,
		NumWidth:     width,
		NumHeight:    height,
		BackIsHidden: true,
		UniqueBack:   false,
		Type:         0,
	}
	return state, nil
}

func buildCardObject(ttsCardID int, cardName string, deckKey int, state CustomDeckState) (CardObject, error) {
	if ttsCardID <= 0 {
		return CardObject{}, fmt.Errorf("card ID cannot be less than or equal to zero: %d", ttsCardID)
	}
	if deckKey <= 0 {
		return CardObject{}, fmt.Errorf("deck key cannot be less than or equal to zero: %d", deckKey)
	}
	if ttsCardID/100 != deckKey {
		return CardObject{}, fmt.Errorf("card ID does not match deck key: %d, %d", ttsCardID, deckKey)
	}
	if cardName == "" {
		return CardObject{}, fmt.Errorf("card name cannot be empty")
	}
	if state.FaceURL == "" {
		return CardObject{}, fmt.Errorf("card face path must not be empty")
	}
	if state.BackURL == "" {
		return CardObject{}, fmt.Errorf("card back path must not be empty")
	}
	key := strconv.Itoa(deckKey)
	card := CardObject{
		Name:     "Card",
		Nickname: cardName,
		CardID:   ttsCardID,
		CustomDeck: map[string]CustomDeckState{
			key: state,
		},
		Transform: Transform{
			ScaleX: 1,
			ScaleY: 1,
			ScaleZ: 1,
		},
	}
	return card, nil
}

func buildCardObjects(
	orderedCardIDs []string,
	physicalDeckIDs []int,
	deckKey int,
	state CustomDeckState,
	repository decks.CardCatalog,
) ([]CardObject, error) {
	if repository == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}
	if len(orderedCardIDs) != len(physicalDeckIDs) {
		return nil, fmt.Errorf("card IDs and deck IDs do not have the same number of values")
	}
	cardObjects := make([]CardObject, 0, len(orderedCardIDs))
	for index := range orderedCardIDs {
		internalID := orderedCardIDs[index]
		ttsID := physicalDeckIDs[index]
		card, found := repository.FindByID(internalID)
		if !found {
			return nil, fmt.Errorf("card %q at position %d was not found", internalID, index+1)
		}
		cardObject, err := buildCardObject(ttsID, card.Name, deckKey, state)
		if err != nil {
			return nil, fmt.Errorf("build card %q at position %d: %w", internalID, index+1, err)
		}
		cardObjects = append(cardObjects, cardObject)
	}
	return cardObjects, nil
}

func buildDeckObject(
	nickname string,
	orderedIDs []string,
	deckKey int,
	faceURL string,
	backURL string,
	transform Transform,
	repository decks.CardCatalog,
) (
	DeckObject,
	[]string,
	error,
) {
	if nickname == "" {
		return DeckObject{}, nil, fmt.Errorf("nickname field cannot be empty")
	}
	if len(orderedIDs) <= 0 {
		return DeckObject{}, nil, fmt.Errorf("deck size must be greater than 0")
	}
	if transform.ScaleX == 0 || transform.ScaleY == 0 || transform.ScaleZ == 0 {
		return DeckObject{}, nil, fmt.Errorf("transform scale factors cannot be zero")
	}
	deckIDs, sheetIDs, err := genDeckIDs(orderedIDs, deckKey)
	if err != nil {
		return DeckObject{}, nil, fmt.Errorf("generate deck IDs: %w", err)
	}
	deckState, err := buildCustomDeckState(faceURL, backURL, len(sheetIDs))
	if err != nil {
		return DeckObject{}, nil, fmt.Errorf("build custom deck state: %w", err)
	}
	cardObjects, err := buildCardObjects(orderedIDs, deckIDs, deckKey, deckState, repository)
	if err != nil {
		return DeckObject{}, nil, fmt.Errorf("build contained cards: %w", err)
	}
	key := strconv.Itoa(deckKey)
	object := DeckObject{
		Name:        "Deck",
		Nickname:    nickname,
		Description: "",
		Transform:   transform,
		DeckIDs:     deckIDs,
		CustomDeck: map[string]CustomDeckState{
			key: deckState,
		},
		ContainedObjects: cardObjects,
	}
	return object, sheetIDs, nil
}

func buildSavedObject(
	deck *decks.Deck,
	mainFace string,
	sideFace string,
	backFace string,
	repository decks.CardCatalog,
) (SavedObject, []string, []string, error) {
	var mainTransform = Transform{
		PosX:   -2.5,
		PosY:   1,
		RotZ:   180,
		ScaleX: 1,
		ScaleY: 1,
		ScaleZ: 1,
	}
	var sideObject DeckObject
	var sideSheetIDs []string
	if deck == nil {
		return SavedObject{}, nil, nil, fmt.Errorf("deck cannot be empty")
	}
	if repository == nil {
		return SavedObject{}, nil, nil, fmt.Errorf("repository cannot be empty")
	}
	mainIDs := cardIDsForImageExport(deck.MainDeck, deck.MainOrder)
	if len(mainIDs) == 0 {
		return SavedObject{}, nil, nil, fmt.Errorf("main deck cannot be empty")
	}
	mainObject, mainSheetIDs, err := buildDeckObject(deck.Name+" - Main Deck", mainIDs, 1, mainFace, backFace, mainTransform, repository)
	if err != nil {
		return SavedObject{}, nil, nil, fmt.Errorf("build main deck object: %w", err)
	}
	objects := make([]DeckObject, 0, 2)
	objects = append(objects, mainObject)

	sideIDs := cardIDsForImageExport(deck.SideDeck, deck.SideOrder)
	if len(sideIDs) > 0 {
		var sideTransform = Transform{
			PosX:   2.5,
			PosY:   1,
			RotZ:   180,
			ScaleX: 1,
			ScaleY: 1,
			ScaleZ: 1,
		}
		sideObject, sideSheetIDs, err = buildDeckObject(deck.Name+" - Sideboard", sideIDs, 2, sideFace, backFace, sideTransform, repository)
		if err != nil {
			return SavedObject{}, nil, nil, fmt.Errorf("build side deck object: %w", err)
		}
		objects = append(objects, sideObject)
	}
	savedObject := SavedObject{
		SaveName:     deck.Name,
		ObjectStates: objects,
	}
	return savedObject, mainSheetIDs, sideSheetIDs, nil
}

func writeSavedObjectJSON(
	writer io.Writer,
	object SavedObject,
) error {
	if writer == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if object.SaveName == "" {
		return fmt.Errorf("save name cannot be empty")
	}
	if len(object.ObjectStates) == 0 {
		return fmt.Errorf("object states cannot be zero")
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(object)
	if err != nil {
		return fmt.Errorf("encode TTS saved object: %w", err)
	}
	return nil
}
