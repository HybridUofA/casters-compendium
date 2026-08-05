package model

type CardID string
type MatchCardID string
type PlayerID string
type CardOrientation string
type Revision uint64

const CasterTokenCardID CardID = "caster-token"

const (
	OrientationRecovered CardOrientation = "Recovered"
	OrientationRested    CardOrientation = "Rested"
	OrientationReversed  CardOrientation = "Reversed"
)

type Phase string

const (
	PhaseRecovery Phase = "Recovery"
	PhaseDraw     Phase = "Draw"
	PhaseCall     Phase = "Call"
	PhaseMain     Phase = "Main"
	PhaseBattle   Phase = "Battle"
	PhaseEnd      Phase = "End"
)

type Status string

const (
	StatusSetup      Status = "Setup"
	StatusInProgress Status = "In Progress"
	StatusFinished   Status = "Finished"
)

type CardCategory string

const (
	CategoryPrintedCard CardCategory = "printed"
	CategoryTokenCard   CardCategory = "token"
)

type CardFace string

const (
	CardFaceUp   CardFace = "faceup"
	CardFaceDown CardFace = "facedown"
)

type CardInstance struct {
	CardID       CardID
	MatchID      MatchCardID
	Owner        PlayerID
	Controller   PlayerID
	CardCategory CardCategory
	Face         CardFace
	Orientation  CardOrientation
}

type Element string

const (
	ElementAes   Element = "Aes"
	ElementAqua  Element = "Aqua"
	ElementIgnus Element = "Ignus"
	ElementLuna  Element = "Luna"
	ElementSilva Element = "Silva"
	ElementSolis Element = "Solis"
	ElementTerra Element = "Terra"
	ElementVoid  Element = "Void"
)

type AetherPool struct {
	Aes          int
	Aqua         int
	Ignus        int
	Luna         int
	Silva        int
	Solis        int
	Terra        int
	Void         int
	NonElemental int
}

type PlayerState struct {
	ID PlayerID
	// Index 0 of PlayerState.Deck is the top of the Deck.
	Deck   []MatchCardID
	Hand   []MatchCardID
	Orbs   []MatchCardID
	Aether AetherPool
	// OpeningHandFinalized is true after the player either keeps their opening
	// hand or completes their one permitted opening-hand replacement.
	OpeningHandFinalized bool
	CasterZone           []MatchCardID
	ServantZone          []MatchCardID
	Graveyard            []MatchCardID
	Exile                []MatchCardID
}

type MatchState struct {
	CardInstances map[MatchCardID]CardInstance
	Players       [2]PlayerState
	FirstPlayer   PlayerID
	MatchStatus   Status
	Revision      Revision
	Turn          TurnState
}

type TurnState struct {
	Number          int
	ActivePlayer    PlayerID
	Phase           Phase
	CallActionTaken bool
}
