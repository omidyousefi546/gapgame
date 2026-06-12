package word_guess

type GuessState string

const (
	StateChoosingType    GuessState = "choosing_type"
	StateWaitingForWord  GuessState = "waiting_for_word" // legacy alias kept for old saved rooms
	StateWaitingSecrets  GuessState = "waiting_secrets"
	StatePlaying         GuessState = "playing"
)

type GameWordGuess struct {
	Type string `json:"type"` // "fa", "en", "num"

	// Legacy single-target fields kept so older serialized rooms can still load.
	TargetWord   string `json:"target_word"`
	DisplayWord  string `json:"display_word"`
	GuesserID    int64  `json:"guesser_id"`
	CreatorID    int64  `json:"creator_id"`
	WrongGuesses []string `json:"wrong_guesses"`
	MaxTries     int      `json:"max_tries"`

	State GuessState `json:"state"`

	// New two-player state. Each player first submits a secret. During play,
	// P1Display/P1Wrong are P1's progress while guessing P2Secret; P2Display/P2Wrong
	// are P2's progress while guessing P1Secret.
	P1Secret string `json:"p1_secret"`
	P2Secret string `json:"p2_secret"`
	P1Ready  bool   `json:"p1_ready"`
	P2Ready  bool   `json:"p2_ready"`

	P1Display string   `json:"p1_display"`
	P2Display string   `json:"p2_display"`
	P1Wrong   []string `json:"p1_wrong"`
	P2Wrong   []string `json:"p2_wrong"`

	CurrentTurn int64 `json:"current_turn"`
}

func (g *GameWordGuess) Reset() {
	g.Type = ""
	g.TargetWord = ""
	g.DisplayWord = ""
	g.GuesserID = 0
	g.CreatorID = 0
	g.State = StateChoosingType
	g.WrongGuesses = nil
	g.MaxTries = 6
	g.P1Secret = ""
	g.P2Secret = ""
	g.P1Ready = false
	g.P2Ready = false
	g.P1Display = ""
	g.P2Display = ""
	g.P1Wrong = nil
	g.P2Wrong = nil
	g.CurrentTurn = 0
}

func (g *GameWordGuess) GameType() string {
	return "gameWordGuess"
}
