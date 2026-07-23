package deckexport

import (
	"fmt"
	"strconv"
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
