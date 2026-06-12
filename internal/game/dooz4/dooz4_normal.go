package dooz4

import (
	"GapGame/internal/game/game_manager"
	"GapGame/internal/utils"
	"fmt"
	"strconv"
	"time"

	tele "gopkg.in/telebot.v3"
)

type GameDooz4Normal struct {
	Board [7][7]int
}

func (g *GameDooz4Normal) Reset() {
	g.Board = [7][7]int{}
}
func (g *GameDooz4Normal) GameType() string {
	return "gameDooz4Normal"
}

func (g *GameDooz4Normal) CheckWin(x, y, markNumber int) bool {

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

func (g *GameDooz4Normal) MakeMove(
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

	if g.Board[y][x] != 0 {
		c.Respond(&tele.CallbackResponse{
			Text: "انتخاب شده است!",
		})

		return false
	}

	markNumber := 1
	mark := "🔵"

	if player.ID == room.Player2.ID {
		markNumber = 2
		mark = "🔴"
	}

	// اضافه کردن منطق جاذبه (پر شدن از پایین)
	found := false
	for i := 0; i < 7; i++ {
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

	if room.Turn == room.Player1.ID {
		room.Turn = room.Player2.ID
		user = room.Player2.FirstName

	} else {
		room.Turn = room.Player1.ID
		user = room.Player1.FirstName
	}

	board := boardDooz4Keyboard(&g.Board)
	msg := fmt.Sprintf("نوبت %v (%v)", user, mark)
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
			}, fmt.Sprintf(utils.GameWinWithName, player.FirstName), boardDooz4KeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(utils.GameLoseWithName, player.FirstName), boardDooz4KeyboardDisabled(&g.Board))
		} else {
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(utils.GameWinWithName, player.FirstName), boardDooz4KeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID1),
				ChatID:    room.Player1.ID,
			}, fmt.Sprintf(utils.GameLoseWithName, player.FirstName), boardDooz4KeyboardDisabled(&g.Board))
		}
		room.Reset()
		return true
	} else {
		return false
	}

}
