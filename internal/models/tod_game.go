package models

import "time"

// TodGame represents a Truth or Dare game session between two players
type TodGame struct {
	ID      uint  `gorm:"primaryKey"`
	MatchID uint  `gorm:"not null;index;uniqueIndex"` // Links to existing Match
	Match   Match `gorm:"foreignKey:MatchID;constraint:OnDelete:CASCADE"`

	// Game State
	State           string `gorm:"type:varchar(30);not null;index;default:'matchmaking'"` // State machine state
	CurrentTurnID   uint   `gorm:"index"`                                                 // Which turn we're on
	ActivePlayerID  uint   `gorm:"index"`                                                 // Player making choice/doing challenge
	PassivePlayerID uint   `gorm:"index"`                                                 // Player judging

	// Round Tracking
	CurrentRound int `gorm:"default:1"`
	MaxRounds    int `gorm:"default:10"` // 0 = unlimited

	// Timing
	TurnStartedAt  *time.Time
	TurnDeadline   *time.Time
	WarningShownAt *time.Time // Track if 30s warning sent

	// Game Settings
	AllowItems      bool   `gorm:"default:true"`
	DifficultyLevel string `gorm:"type:varchar(20);default:'normal'"` // normal, hard

	// Metadata
	StartedAt *time.Time
	EndedAt   *time.Time
	WinnerID  *uint
	EndReason string `gorm:"type:varchar(50)"` // completed, forfeit, quit, afk

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// State constants
const (
	TodStateMatchmaking     = "matchmaking"
	TodStateCoinFlip        = "coin_flip"
	TodStateWaitingChoice   = "waiting_choice"
	TodStateWaitingProof    = "waiting_proof"
	TodStateWaitingJudgment = "waiting_judgment"
	TodStateForfeit         = "forfeit"
	TodStateGameEnd         = "game_end"
)

// TodTurn represents a single turn in the game
type TodTurn struct {
	ID     uint    `gorm:"primaryKey"`
	GameID uint    `gorm:"not null;index"`
	Game   TodGame `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`

	RoundNumber int  `gorm:"not null"`
	PlayerID    uint `gorm:"not null;index"` // Active player
	JudgeID     uint `gorm:"not null;index"` // Passive player

	// Choice Phase
	Choice   string `gorm:"type:varchar(10)"` // truth, dare, item
	ChosenAt *time.Time

	// Challenge Phase
	ChallengeID   *uint
	Challenge     *TodChallenge `gorm:"foreignKey:ChallengeID"`
	ChallengeText string        `gorm:"type:text"` // Cached for history

	// Proof Phase
	ProofType        string `gorm:"type:varchar(20)"` // text, voice, image, video
	ProofData        string `gorm:"type:text"`        // File ID or text content
	ProofSubmittedAt *time.Time

	// Judgment Phase
	JudgmentResult string `gorm:"type:varchar(20)"` // accepted, rejected, timeout
	JudgmentReason string `gorm:"type:text"`
	JudgedAt       *time.Time

	// Item Usage
	ItemUsed   string `gorm:"type:varchar(20)"` // shield, swap, mirror
	ItemUsedAt *time.Time

	// Rewards
	XPAwarded    int `gorm:"default:0"`
	CoinsAwarded int `gorm:"default:0"`

	// Timing
	StartedAt   time.Time `gorm:"autoCreateTime"`
	CompletedAt *time.Time
	TimeoutAt   *time.Time
}

// TodChallenge represents a truth or dare challenge
type TodChallenge struct {
	ID   uint   `gorm:"primaryKey"`
	Type string `gorm:"type:varchar(10);not null;index"` // truth, dare
	Text string `gorm:"type:text;not null"`

	// Categorization
	Difficulty    string `gorm:"type:varchar(20);index"` // easy, medium, hard
	Category      string `gorm:"type:varchar(50);index"` // funny, romantic, hot, embarrassing
	GenderTarget  string `gorm:"type:varchar(10);index"` // male, female, all
	RelationLevel string `gorm:"type:varchar(20);index"` // stranger, friend, close

	// Proof Requirements
	ProofType string `gorm:"type:varchar(20);not null"` // text, voice, image, video, none
	ProofHint string `gorm:"type:text"`                 // Hint for what proof to provide

	// Rewards
	XPReward   int `gorm:"default:20"`
	CoinReward int `gorm:"default:10"`

	// Stats
	TimesUsed      int     `gorm:"default:0"`
	AcceptanceRate float64 `gorm:"default:0"` // % of times accepted by judge

	// Metadata
	IsActive  bool      `gorm:"default:true;index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// Challenge type constants
const (
	TodTypeTruth = "truth"
	TodTypeDare  = "dare"
)

// Proof type constants
const (
	ProofTypeText  = "text"
	ProofTypeVoice = "voice"
	ProofTypeImage = "image"
	ProofTypeVideo = "video"
	ProofTypeNone  = "none"
)

// Item type constants
const (
	ItemTypeShield = "shield"
	ItemTypeSwap   = "swap"
	ItemTypeMirror = "mirror"
)

// TodPlayerStats tracks player statistics
type TodPlayerStats struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;uniqueIndex"`
	User   User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	// Game Stats
	GamesPlayed int `gorm:"default:0"`
	GamesWon    int `gorm:"default:0"`
	GamesLost   int `gorm:"default:0"`

	// Challenge Stats
	TruthsChosen        int `gorm:"default:0"`
	DaresChosen         int `gorm:"default:0"`
	ChallengesCompleted int `gorm:"default:0"`
	ChallengesFailed    int `gorm:"default:0"`

	// Judge Stats
	JudgmentsMade       int     `gorm:"default:0"`
	JudgmentsAccepted   int     `gorm:"default:0"`
	JudgmentsRejected   int     `gorm:"default:0"`
	JudgeScore          float64 `gorm:"default:100.0"` // 0-100, starts at 100
	UnfairJudgmentCount int     `gorm:"default:0"`     // Strikes for unfair judging

	// Item Stats
	ItemsUsed    int `gorm:"default:0"`
	ShieldsOwned int `gorm:"default:1"` // Starting inventory
	SwapsOwned   int `gorm:"default:1"`
	MirrorsOwned int `gorm:"default:1"`

	// Timing Stats
	AvgResponseTime int `gorm:"default:0"` // Seconds
	TimeoutCount    int `gorm:"default:0"`

	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TodJudgmentLog tracks all judgments for anti-abuse
type TodJudgmentLog struct {
	ID     uint    `gorm:"primaryKey"`
	TurnID uint    `gorm:"not null;index"`
	Turn   TodTurn `gorm:"foreignKey:TurnID;constraint:OnDelete:CASCADE"`

	JudgeID  uint `gorm:"not null;index"`
	Judge    User `gorm:"foreignKey:JudgeID"`
	PlayerID uint `gorm:"not null;index"`
	Player   User `gorm:"foreignKey:PlayerID"`

	Result       string `gorm:"type:varchar(20);not null"` // accepted, rejected
	ProofQuality int    `gorm:"default:0"`                 // 1-5 rating (future feature)

	// Anti-abuse tracking
	IsSuspicious    bool   `gorm:"default:false;index"`
	SuspicionReason string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TodActionLog tracks all actions for idempotency
type TodActionLog struct {
	ID       uint   `gorm:"primaryKey"`
	GameID   uint   `gorm:"not null;index"`
	UserID   uint   `gorm:"not null;index"`
	ActionID string `gorm:"type:varchar(100);not null;uniqueIndex"` // UUID for deduplication
	Action   string `gorm:"type:varchar(50);not null"`              // choice, proof, judgment, etc.

	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (TodGame) TableName() string {
	return "tod_games"
}

func (TodTurn) TableName() string {
	return "tod_turns"
}

func (TodChallenge) TableName() string {
	return "tod_challenges"
}

func (TodPlayerStats) TableName() string {
	return "tod_player_stats"
}

func (TodJudgmentLog) TableName() string {
	return "tod_judgment_logs"
}

func (TodActionLog) TableName() string {
	return "tod_action_logs"
}
