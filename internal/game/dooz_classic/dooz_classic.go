package dooz_classic

import (
	"GapGame/internal/game/game_manager"
	"GapGame/pkg/messages"
	"fmt"
	"strconv"
	"time"

	tele "gopkg.in/telebot.v3"
)

type GameDoozClassic struct {
	Board [3][3]int
}

func (g *GameDoozClassic) Reset() {
	g.Board = [3][3]int{}
}
func (g *GameDoozClassic) GameType() string {
	return "gameDoozClassic"
}

func (g *GameDoozClassic) CheckWin(x, y, markNumber int) bool {

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

		for i >= 0 && i < 3 && j >= 0 && j < 3 && g.Board[j][i] == markNumber {
			count++
			i += dx
			j += dy
		}

		// backward
		i = x - dx
		j = y - dy

		for i >= 0 && i < 3 && j >= 0 && j < 3 && g.Board[j][i] == markNumber {
			count++
			i -= dx
			j -= dy
		}
		if count >= 3 {
			return true
		}
	}

	return false
}

func (g *GameDoozClassic) MakeMove(
	b *tele.Bot,
	c tele.Context,
	boardDoozClassicKeyboard func(Board *[3][3]int) *tele.ReplyMarkup,
	boardDoozClassicKeyboardDisabled func(Board *[3][3]int) *tele.ReplyMarkup,
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

	if player.ID == room.Player2.ID {
		markNumber = 2
	}

	g.Board[y][x] = markNumber
	var user string

	nextMark := "🔵"
	if room.Turn == room.Player1.ID {
		room.Turn = room.Player2.ID
		user = room.Player2.FirstName
		nextMark = "🔴"

	} else {
		room.Turn = room.Player1.ID
		user = room.Player1.FirstName
		nextMark = "🔵"
	}

	board := boardDoozClassicKeyboard(&g.Board)
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
			}, fmt.Sprintf(messages.GameWinWithName, player.FirstName), boardDoozClassicKeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(messages.GameLoseWithName, player.FirstName), boardDoozClassicKeyboardDisabled(&g.Board))
		} else {
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID2),
				ChatID:    room.Player2.ID,
			}, fmt.Sprintf(messages.GameWinWithName, player.FirstName), boardDoozClassicKeyboardDisabled(&g.Board))
			b.Edit(&tele.StoredMessage{
				MessageID: strconv.Itoa(room.MsgID1),
				ChatID:    room.Player1.ID,
			}, fmt.Sprintf(messages.GameLoseWithName, player.FirstName), boardDoozClassicKeyboardDisabled(&g.Board))
		}
		room.Reset()
		return true
	} else {
		return false
	}

}
