package dooz4

import (
	"GapGame/internal/game/game_manager"
	"GapGame/pkg/messages"
	"fmt"
	"strconv"
	"time"

	tele "gopkg.in/telebot.v3"
)

type GameDooz4 struct {
	Board [7][7]int
}

func (g *GameDooz4) Reset() {
	g.Board = [7][7]int{}
}
func (g *GameDooz4) GameType() string {
	return "gameDooz4Gravity"
}

func (g *GameDooz4) IsDraw() bool {
	for _, row := range g.Board {
		for _, cell := range row {
			if cell == 0 {
				return false
			}
		}
	}
	return true
}

func (g *GameDooz4) CheckWin(x, y, markNumber int) bool {

	directions := [][]int{
		{0, 1},  // horizontal
		{1, 0},  // vertical
		{1, 1},  // diagonal \
		{1, -1}, // diagonal /
	}

	for _, d := range directions {

		count := 1

		dx := d[0]
		dy := d[1]

		// forward
		i := x + dx
		j := y + dy

		for i >= 0 && i < 7 && j >= 0 && j < 7 && g.Board[j][i] == markNumber {
			count++
			i += dx
			j += dy
		}
		// backward
		i = x - dx
		j = y - dy

		for i >= 0 && i < 7 && j >= 0 && j < 7 && g.Board[j][i] == markNumber {
			count++
			i -= dx
			j -= dy
		}
		if count >= 4 {
			return true
		}
	}

	return false
}

func (g *GameDooz4) MakeMove(
	b *tele.Bot,
	c tele.Context,
	boardDooz4Keyboard func(Board *[7][7]int) *tele.ReplyMarkup,
	boardDooz4KeyboardDisabled func(Board *[7][7]int) *tele.ReplyMarkup,
	room *game_manager.Room,
	player *tele.User,
	x, y int,
) bool {

	room.LastMove = time.Now()
	if room.Turn != player.ID {
		c.Respond(&tele.CallbackResponse{
			Text: "نوبت شما نیست!",
		})
		return false
	}


	markNumber := 1

	if player.ID == room.Player2.ID {
		markNumber = 2
	}

	found := false
	for i := range g.Board[y] {
		if g.Board[y][6-i] == 0 {
			x = 6 - i
			g.Board[y][x] = markNumber
			found = true
			break
		}
	}

	if !found {
		c.Respond(&tele.CallbackResponse{
			Text: "این ستون پر شده است!",
		})
		return false
	}

	var user string
	var nextMark string

	if room.Turn == room.Player1.ID {
		room.Turn = room.Player2.ID
		user = room.NameFor(room.Player2.ID)
		nextMark = "🔵"
	} else {
		room.Turn = room.Player1.ID
		user = room.NameFor(room.Player1.ID)
		nextMark = "🔴"
	}

	board := boardDooz4Keyboard(&g.Board)
	msg := fmt.Sprintf("نوبت %v (%v)", user, nextMark)
	b.Edit(&tele.StoredMessage{
		MessageID: strconv.Itoa(room.MsgID1),
		ChatID:    room.Player1.ID,
	}, msg, board)
	b.Edit(&tele.StoredMessage{
		MessageID: strconv.Itoa(room.MsgID2),
		ChatID:    room.Player2.ID,
	}, msg, board)

	if g.CheckWin(x, y, markNumber) {
		if player.ID == room.Player1.ID {
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID1),
				ChatID:    room.Player1.ID,
			}, fmt.Sprintf(messages.GameWinWithName, room.NameFor(room.Player1.ID)), boardDooz4KeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(messages.GameLoseWithName, room.NameFor(room.Player1.ID)), boardDooz4KeyboardDisabled(&g.Board))
		} else {
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(messages.GameWinWithName, room.NameFor(room.Player2.ID)), boardDooz4KeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID1),
				ChatID:    room.Player1.ID,
			}, fmt.Sprintf(messages.GameLoseWithName, room.NameFor(room.Player2.ID)), boardDooz4KeyboardDisabled(&g.Board))
		}
		room.Reset()
		return true
	} else if g.IsDraw() {
		drawMsg := "🤝 بازی مساوی شد! همه خانه‌ها پر شدند و هیچ بازیکنی برنده نشد."
		b.Edit(&tele.StoredMessage{
			MessageID: strconv.Itoa(room.MsgID1),
			ChatID:    room.Player1.ID,
		}, drawMsg, boardDooz4KeyboardDisabled(&g.Board))
		b.Edit(&tele.StoredMessage{
			MessageID: strconv.Itoa(room.MsgID2),
			ChatID:    room.Player2.ID,
		}, drawMsg, boardDooz4KeyboardDisabled(&g.Board))
		room.Reset()
		return true
	} else {
		return false
	}

}
