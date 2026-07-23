package deckexport

import "fmt"

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
