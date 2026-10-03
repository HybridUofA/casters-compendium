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
	Stock        []MatchCardID
}

type CardLocation struct {
	PlayerIndex int
	Zone        Zone
	CardIndex   int
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

type AetherPayment struct {
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

// CastPaymentPlan records the Aether sources a player wants to use as part of
// one atomic cast. Payment may also spend Aether that was already in the pool.
type CastPaymentPlan struct {
	SourceCardIDs []MatchCardID
	Payment       AetherPayment
}

type ChaseLinkID uint64
type ChaseLinkKind string

const (
	ChaseLinkCardPlay         ChaseLinkKind = "card_play"
	ChaseLinkActivatedAbility ChaseLinkKind = "activated_ability"
	ChaseLinkTriggeredAbility ChaseLinkKind = "triggered_ability"
)

type ChaseLink struct {
	ID               ChaseLinkID
	Controller       PlayerID
	SourceCardID     MatchCardID
	Kind             ChaseLinkKind
	EntryOrientation CardOrientation
}

type Chase []ChaseLink

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
	CardInstances        map[MatchCardID]CardInstance
	Players              [2]PlayerState
	FirstPlayer          PlayerID
	MatchStatus          Status
	Revision             Revision
	Turn                 TurnState
	ChaseLinks           Chase
	PriorityHolder       PlayerID
	PassCount            int
	NextLinkID           ChaseLinkID
	PrioritySequenceOpen bool
	Result               MatchResult
	Attack               AttackState
}

type TurnState struct {
	Number          int
	ActivePlayer    PlayerID
	Phase           Phase
	CallActionTaken bool
}

type Zone string

const (
	ZoneHand      Zone = "Hand"
	ZoneCaster    Zone = "CasterZone"
	ZoneServant   Zone = "ServantZone"
	ZoneDeck      Zone = "Deck"
	ZoneGraveyard Zone = "Graveyard"
	ZoneExile     Zone = "Exile"
	ZoneOrbs      Zone = "Orbs"
)

type MoveCardCommand struct {
	CardID              MatchCardID
	DestinationPlayerID PlayerID
	DestinationZone     Zone
	DestinationFace     CardFace
	EntryOrientation    CardOrientation
	Placement           DeckPlacement
}

type DeckPlacement string

const (
	DeckPlacementTop    DeckPlacement = "top"
	DeckPlacementBottom DeckPlacement = "bottom"
)

type MatchEndReason string

const (
	EndReasonDeckOut          MatchEndReason = "Deck Out"
	EndReasonZeroOrbs         MatchEndReason = "Zero Orbs"
	EndReasonSimultaneousLoss MatchEndReason = "Simultaneous Loss"
)

type MatchResult struct {
	Winner PlayerID
	Loser  PlayerID
	IsDraw bool
	Reason MatchEndReason
}

type BattleStep string

const (
	BattleStepIdle             BattleStep = ""
	BattleStepDeclared         BattleStep = "Declared"
	BattleStepAwaitingJudgment BattleStep = "Awaiting Judgment"
)

type AttackTargetKind string

const (
	AttackTargetPlayer  AttackTargetKind = "Player"
	AttackTargetServant AttackTargetKind = "Servant"
)

type AttackState struct {
	AttackerID   MatchCardID
	TargetKind   AttackTargetKind
	TargetCardID MatchCardID
	Step         BattleStep
}
