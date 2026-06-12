package rps

type GameRPS struct {
	P1Move string `json:"p1_move"`
	P2Move string `json:"p2_move"`
}

func (g *GameRPS) Reset() {
	g.P1Move = ""
	g.P2Move = ""
}

func (g *GameRPS) GameType() string {
	return "gameRPS"
}
