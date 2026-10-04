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
	// GrantedDoubleCorrupt is a manual marker for effects that give a Servant
	// Double Corrupt (caster abilities, temporary grants). Cleared when the
	// card leaves the Servant zone.
	GrantedDoubleCorrupt bool
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
	// PendingDraw tracks an optional replace-on-draw choice (Sage Advice).
	PendingDraw PendingDraw
	// PendingBreak tracks optional Break plays after Orb corruption.
	// CardIDs[0] is the card currently offered; the owner may Use or Decline
	// each entry in order (multi-corrupt queues several at once).
	PendingBreak PendingBreak
	// KnownCards records MatchCardIDs whose faces a player has seen and should
	// keep seeing while those cards remain in a hidden zone (e.g. peeked Orbs).
	KnownCards map[PlayerID]map[MatchCardID]struct{}
}

// PendingBreak is set when corrupted Orbs with Break enter their owner's hand.
type PendingBreak struct {
	PlayerID PlayerID
	CardIDs  []MatchCardID
}

// PendingDrawStep is the replace-on-draw decision state.
type PendingDrawStep string

const (
	PendingDrawIdle  PendingDrawStep = ""
	PendingDrawOffer PendingDrawStep = "Offer"
	PendingDrawDig   PendingDrawStep = "Dig"
)

// PendingDraw is set when a player would draw and may apply Sage Advice instead.
type PendingDraw struct {
	PlayerID  PlayerID
	Remaining int
	Step      PendingDrawStep
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
	// CorruptCount is how many enemy Orbs the attacker chooses after a
	// player-attack judgment (1 normally, up to 2 with Double Corrupt,
	// capped by remaining Orbs). Meaningful only while Step is
	// AwaitingJudgment.
	CorruptCount int
}
