package word_guess

type GuessState string

const (
	StateChoosingType   GuessState = "choosing_type"
	StateWaitingForWord GuessState = "waiting_for_word"
	StatePlaying        GuessState = "playing"
)

type GameWordGuess struct {
	Type         string     `json:"type"`         // "fa", "en", "num"
	TargetWord   string     `json:"target_word"`  // کلمه‌ای که باید حدس زده شود
	DisplayWord  string     `json:"display_word"` // کلمه به صورت _ _ _
	GuesserID    int64      `json:"guesser_id"`   // کسی که باید حدس بزند
	CreatorID    int64      `json:"creator_id"`   // کسی که کلمه را تعیین کرده
	State        GuessState `json:"state"`
	WrongGuesses []string   `json:"wrong_guesses"`
	MaxTries     int        `json:"max_tries"`
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
}

func (g *GameWordGuess) GameType() string {
	return "gameWordGuess"
}
