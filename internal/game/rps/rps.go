package rps

type GameRPS struct {
	P1Move  string `json:"p1_move"`
	P2Move  string `json:"p2_move"`
	P1Score int    `json:"p1_score"`
	P2Score int    `json:"p2_score"`
	Round   int    `json:"round"`
}

func (g *GameRPS) Reset() {
	g.P1Move = ""
	g.P2Move = ""
	g.P1Score = 0
	g.P2Score = 0
	g.Round = 1
}

func (g *GameRPS) GameType() string {
	return "gameRPS"
}
