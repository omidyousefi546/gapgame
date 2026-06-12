// internal/bot/game_handlers.go
package bot

import (
	"GapGame/internal/game/dare_and_truth"
	"GapGame/internal/game/dooz4"
	"GapGame/internal/game/dooz_classic"
	"GapGame/internal/game/game_manager"
	"GapGame/internal/game/rps"
	"GapGame/internal/game/word_guess"
	"GapGame/internal/session"
	"GapGame/internal/utils"
	"GapGame/pkg/messages"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) showGamesHandler(c tele.Context) error {
	return editOrSend(c, messages.PickGame, GameMenuKeyboard())
}

func (h *Handler) selectGameHandler(c tele.Context) error {
	gameSelected := c.Data()

	var state game_manager.GameState
	var gameType string

	switch gameSelected {
	case "game_dooz4_normal":
		state = &dooz4.GameDooz4Normal{}
		gameType = "بازی دوز ۴ عادی"
	case "game_dooz4_gravity":
		state = &dooz4.GameDooz4{}
		gameType = "بازی دوز ۴ جاذبه"
	case "game_dooz_classic":
		state = &dooz_classic.GameDoozClassic{}
		gameType = "بازی دوز کلاسیک"
	case "game_dare_and_truth":
		state = &dare_and_truth.GameDareTruth{}
		gameType = "جرات حقیقت"
	default:
		return c.Respond(&tele.CallbackResponse{Text: messages.InvalidGame})
	}

	h.rooms.RemoveRoomsByPlayerID(c.Sender().ID)
	room := h.rooms.CreateRoom(c.Sender(), state)

	if err := h.redis.SetUserState(c.Sender().ID, session.StateInGame); err != nil {
		return err
	}

	link := fmt.Sprintf(
		"[ورود به بازی 🔗](https://ble.ir/%s?start=join_%d_%v)",
		messages.BOT_USERNAME, c.Sender().ID, state.GameType(),
	)
	msgText := fmt.Sprintf(messages.CreateGame, gameType, link)

	msg, err := h.bot.Send(c.Sender(), msgText, &tele.SendOptions{ParseMode: tele.ModeMarkdown}, InGameMenuKeyboard())
	if err != nil {
		return err
	}
	room.MsgID1 = msg.ID
	room.MsgID2 = msg.ID
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) finishGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.ReStartMessage, MainMenuKeyboard())
	}
	return editOrSend(c, messages.ReConfirmFinish, ConfirmFinishGameMenuKeyboard())
}

func (h *Handler) declineFinishGame(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.ReStartMessage, MainMenuKeyboard())
	}
	msg := c.Callback().Message
	if msg != nil {
		return c.Bot().Delete(msg)
	}
	return nil
}

func (h *Handler) confirmFinishGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.ReStartMessage, MainMenuKeyboard())
	}

	var opponent *tele.User
	if c.Sender().ID == room.Player1.ID {
		opponent = room.Player2
	} else {
		opponent = room.Player1
	}

	h.redis.ClearUserState(c.Sender().ID)
	if opponent != nil {
		h.redis.ClearUserState(opponent.ID)
		h.bot.Send(opponent, messages.GameEndedByOpponent, MainMenuKeyboard())
	}

	h.rooms.RemoveRoomByRoomID(room.ID)
	return editOrSend(c, messages.GameEndedByYou, MainMenuKeyboard())
}

func (h *Handler) repeatGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.GameCancelledOpponent, MainMenuKeyboard())
	}
	if room.Player2 == nil {
		return editOrSend(c, messages.GameCancelledOpponent, MainMenuKeyboard())
	}

	if c.Sender().ID != room.Player1.ID {
		room.Player2 = room.Player1
		room.Player1 = c.Sender()
	}
	room.Turn = room.Player1.ID
	room.State.Reset()

	var board *tele.ReplyMarkup
	switch g := room.State.(type) {
	case *dooz4.GameDooz4:
		board = boardDooz4Keyboard(&g.Board, "game_dooz4_gravity")
	case *dooz4.GameDooz4Normal:
		board = boardDooz4Keyboard(&g.Board, "game_dooz4_normal")
	case *dooz_classic.GameDoozClassic:
		board = boardDoozClassicKeyboard(&g.Board)
	case *dare_and_truth.GameDareTruth:
		board = boardDareAndTruthKeyboard()
	default:
		return editOrSend(c, messages.GameCancelledOpponent, MainMenuKeyboard())
	}

	text := fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v (%v)", room.Player1.FirstName, "🔴")
	h.bot.Send(room.Player1, messages.GameDooz4ReStarted, InGameMenuKeyboard())
	h.bot.Send(room.Player2, messages.GameDooz4ReStarted, InGameMenuKeyboard())

	msg1, _ := h.bot.Send(room.Player1, text, board)
	room.MsgID1 = msg1.ID
	msg2, _ := h.bot.Send(room.Player2, text, board)
	room.MsgID2 = msg2.ID
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) cancelGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.GameCancelledOpponent, MainMenuKeyboard())
	}
	h.redis.ClearUserState(c.Sender().ID)
	h.rooms.RemoveRoomByRoomID(room.ID)
	return editOrSend(c, messages.GameCancelledYou, MainMenuKeyboard())
}

func (h *Handler) handleGameJoin(c tele.Context, payload string) error {
	parts := strings.Split(payload, "_")
	if len(parts) < 3 {
		return editOrSend(c, messages.GameInviteLink, MainMenuKeyboard())
	}
	creatorID, _ := strconv.ParseInt(parts[1], 10, 64)
	if c.Sender().ID == creatorID {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameSelfPlay})
	}
	room := h.rooms.GetRoomByPlayerID(creatorID)
	if room == nil || room.Player2 != nil {
		return editOrSend(c, messages.GameUnavailable, MainMenuKeyboard())
	}
	room, _ = h.rooms.JoinRoom(room.ID, c.Sender())
	h.redis.SetUserState(c.Sender().ID, session.StateInGame)

	var msg string
	var keyboard *tele.ReplyMarkup
	switch parts[2] {
	case "gameDooz4Gravity":
		msg = messages.GameDooz4Started
		keyboard = boardDooz4Keyboard(&room.State.(*dooz4.GameDooz4).Board, "game_dooz4_gravity")
	case "gameDooz4Normal":
		msg = messages.GameDooz4Started
		keyboard = boardDooz4Keyboard(&room.State.(*dooz4.GameDooz4Normal).Board, "game_dooz4_normal")
	case "gameDoozClassic":
		msg = messages.GameDoozClassicStarted
		keyboard = boardDoozClassicKeyboard(&room.State.(*dooz_classic.GameDoozClassic).Board)
	case "gameDareAndTruth":
		msg = messages.GameDareAndTruthStarted
		keyboard = boardDareAndTruthKeyboard()
	}

	h.bot.Send(room.Player1, msg, InGameMenuKeyboard())
	h.bot.Send(room.Player2, msg, InGameMenuKeyboard())
	room.StartRoom(h.bot, keyboard)
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDooz4GravityHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz4.GameDooz4).MakeMove(h.bot, c, func(b *[7][7]int) *tele.ReplyMarkup { return boardDooz4Keyboard(b, "game_dooz4_gravity") }, boardDooz4KeyboardDisabled, room, c.Sender(), x, y) {
		h.bot.Send(room.Player1, messages.GameRepeat, AfterGameMenuKeyboard())
		h.bot.Send(room.Player2, messages.GameRepeat, AfterGameMenuKeyboard())
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDooz4NormalHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz4.GameDooz4Normal).MakeMove(h.bot, c, func(b *[7][7]int) *tele.ReplyMarkup { return boardDooz4Keyboard(b, "game_dooz4_normal") }, boardDooz4KeyboardDisabled, room, c.Sender(), x, y) {
		h.bot.Send(room.Player1, messages.GameRepeat, AfterGameMenuKeyboard())
		h.bot.Send(room.Player2, messages.GameRepeat, AfterGameMenuKeyboard())
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDoozClassicHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz_classic.GameDoozClassic).MakeMove(h.bot, c, boardDoozClassicKeyboard, boardDoozClassicKeyboardDisabled, room, c.Sender(), x, y) {
		h.bot.Send(room.Player1, messages.GameRepeat, AfterGameMenuKeyboard())
		h.bot.Send(room.Player2, messages.GameRepeat, AfterGameMenuKeyboard())
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDareAndTruthHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	room.State.(*dare_and_truth.GameDareTruth).MakeMove(h.bot, c, boardDareAndTruthKeyboard(), room, c.Sender(), c.Callback().Data)
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) ChatGameHandler(c tele.Context) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, err := h.redis.GetActiveChat(ctx, c.Sender().ID)
	if err != nil || cs == nil {
		return editOrSend(c, messages.GameNotInChat)
	}
	return editOrSend(c, messages.ChatGameList, ChatGameMenuKeyboard())
}

func (h *Handler) ChatGameRequestCallback(c tele.Context) error {
	gameType := c.Callback().Data
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, _ := h.redis.GetActiveChat(ctx, c.Sender().ID)
	partnerID := cs.User1ID
	if partnerID == c.Sender().ID {
		partnerID = cs.User2ID
	}
	gameName := map[string]string{"rps": "سنگ کاغذ قیچی", "word": "حدس کلمه", "dooz4": "دوز ۴ تایی", "dooz3": "دوز کلاسیک"}[gameType]
	h.bot.Delete(c.Message())
	editOrSend(c, fmt.Sprintf(messages.GameRequestSent, gameName))
	h.bot.Send(&tele.User{ID: partnerID}, fmt.Sprintf(messages.GameInvite, gameName), &tele.SendOptions{ParseMode: tele.ModeMarkdown}, ChatGameRequestKeyboard(gameType))
	return nil
}

func (h *Handler) ChatGameRejectCallback(c tele.Context) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, _ := h.redis.GetActiveChat(ctx, c.Sender().ID)
	partnerID := cs.User1ID
	if partnerID == c.Sender().ID {
		partnerID = cs.User2ID
	}
	h.bot.Delete(c.Message())
	h.bot.Send(&tele.User{ID: partnerID}, messages.GameRequestDenied)
	return editOrSend(c, messages.GameDenied)
}

func (h *Handler) ChatGameAcceptCallback(c tele.Context) error {
	gameType := c.Callback().Data
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, _ := h.redis.GetActiveChat(ctx, c.Sender().ID)
	partnerID := cs.User1ID
	if partnerID == c.Sender().ID {
		partnerID = cs.User2ID
	}
	var state game_manager.GameState
	var kb *tele.ReplyMarkup
	var startMsg string
	switch gameType {
	case "rps":
		state = &rps.GameRPS{}
		kb = boardRPSKeyboard()
		startMsg = "🎮 سنگ کاغذ قیچی شروع شد!"
	case "word":
		state = &word_guess.GameWordGuess{State: word_guess.StateChoosingType}
		kb = boardWordTypeKeyboard()
		startMsg = "🎮 حدس کلمه شروع شد! نوع کلمه را انتخاب کنید:"
	case "dooz4":
		state = &dooz4.GameDooz4Normal{}
		kb = boardDooz4Keyboard(&[7][7]int{}, "game_dooz4_normal")
		startMsg = messages.GameDooz4Started
	case "dooz3":
		state = &dooz_classic.GameDoozClassic{}
		kb = boardDoozClassicKeyboard(&[3][3]int{})
		startMsg = messages.GameDoozClassicStarted
	}
	h.rooms.RemoveRoomsByPlayerID(c.Sender().ID)
	h.rooms.RemoveRoomsByPlayerID(partnerID)
	room := h.rooms.CreateRoom(c.Sender(), state)
	h.rooms.JoinRoom(room.ID, &tele.User{ID: partnerID})
	h.bot.Send(&tele.User{ID: partnerID}, startMsg, kb)
	editOrSend(c, startMsg, kb)
	h.rooms.SaveRoom(room)
	return nil
}

func boardRPSKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(m.Data("🪨", "move_rps", "rock"), m.Data("🧻", "move_rps", "paper"), m.Data("✂️", "move_rps", "scissors")))
	return m
}

func (h *Handler) moveRPSHandler(c tele.Context) error {
	move := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	game := room.State.(*rps.GameRPS)
	if c.Sender().ID == room.Player1.ID {
		if game.P1Move != "" {
			return nil
		}
		game.P1Move = move
	} else {
		if game.P2Move != "" {
			return nil
		}
		game.P2Move = move
	}
	if game.P1Move != "" && game.P2Move != "" {
		res := determineRPSWinner(game.P1Move, game.P2Move)
		msg := fmt.Sprintf("🏁 نتیجه:\nپ۱: %s\nپ۲: %s\n%s", rpsEmoji(game.P1Move), rpsEmoji(game.P2Move), res)
		h.bot.Send(room.Player1, msg)
		h.bot.Send(room.Player2, msg)
		h.rooms.RemoveRoomByRoomID(room.ID)
	} else {
		c.Edit(messages.GameMoveRegistered)
	}
	h.rooms.SaveRoom(room)
	return nil
}

func determineRPSWinner(p1, p2 string) string {
	if p1 == p2 {
		return "🤝 مساوی!"
	}
	if map[string]string{"rock": "scissors", "paper": "rock", "scissors": "paper"}[p1] == p2 {
		return "🎉 پ۱ برد!"
	}
	return "🎉 پ۲ برد!"
}

func rpsEmoji(m string) string {
	return map[string]string{"rock": "🪨", "paper": "🧻", "scissors": "✂️"}[m]
}

func boardWordTypeKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(m.Data("🇮🇷", "word_type", "fa"), m.Data("🇺🇸", "word_type", "en"), m.Data("🔢", "word_type", "num")))
	return m
}

func (h *Handler) wordTypeHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	game := room.State.(*word_guess.GameWordGuess)
	game.Type = c.Callback().Data
	game.State = word_guess.StateWaitingForWord
	game.CreatorID = c.Sender().ID
	partnerID := room.Player1.ID
	if partnerID == c.Sender().ID {
		partnerID = room.Player2.ID
	}
	game.GuesserID = partnerID
	h.bot.Edit(c.Message(), "✅ کلمه را بنویس (۳-۶ حرف):")
	h.bot.Send(&tele.User{ID: partnerID}, "منتظر حریف...")
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) wordGuessMoveHandler(c tele.Context) error {
	char := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	game := room.State.(*word_guess.GameWordGuess)
	if c.Sender().ID != game.GuesserID || strings.Contains(game.DisplayWord, char) {
		return nil
	}
	for _, w := range game.WrongGuesses {
		if w == char {
			return nil
		}
	}
	found := false
	newDisplay := ""
	for _, r := range []rune(game.TargetWord) {
		if string(r) == char || strings.Contains(game.DisplayWord, string(r)) {
			newDisplay += string(r) + " "
			if string(r) == char {
				found = true
			}
		} else {
			newDisplay += "_ "
		}
	}
	game.DisplayWord = strings.TrimSpace(newDisplay)
	if !found {
		game.WrongGuesses = append(game.WrongGuesses, char)
		game.MaxTries--
	}
	h.rooms.SaveRoom(room)
	partnerID := room.Player1.ID
	if partnerID == c.Sender().ID {
		partnerID = room.Player2.ID
	}
	if !strings.Contains(game.DisplayWord, "_") {
		h.bot.Send(c.Sender(), messages.GameYouWon)
		h.bot.Send(&tele.User{ID: partnerID}, messages.GameYouLost)
		h.rooms.RemoveRoomByRoomID(room.ID)
		return nil
	}
	if game.MaxTries <= 0 {
		h.bot.Send(c.Sender(), messages.GameYouLost)
		h.bot.Send(&tele.User{ID: partnerID}, messages.GameYouWon)
		h.rooms.RemoveRoomByRoomID(room.ID)
		return nil
	}
	msg := fmt.Sprintf("کلمه: %s\nفرصت: %d\nخطاها: %s", game.DisplayWord, game.MaxTries, strings.Join(game.WrongGuesses, ","))
	c.Edit(msg, boardWordGuessKeyboard(game))
	h.bot.Send(&tele.User{ID: partnerID}, msg)
	return nil
}

func boardWordGuessKeyboard(game *word_guess.GameWordGuess) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	var chars []string
	if game.Type == "fa" {
		chars = []string{"آ", "ا", "ب", "پ", "ت", "ث", "ج", "چ", "ح", "خ", "د", "ذ", "ر", "ز", "ژ", "س", "ش", "ص", "ض", "ط", "ظ", "ع", "غ", "ف", "ق", "ک", "گ", "ل", "م", "ن", "و", "ه", "ی"}
	} else if game.Type == "en" {
		chars = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	} else {
		chars = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	}
	var rows []tele.Row
	var cur tele.Row
	for i, c := range chars {
		lbl := c
		for _, w := range game.WrongGuesses {
			if w == c {
				lbl = "✖️"
			}
		}
		cur = append(cur, m.Data(lbl, "word_guess", c))
		if (i+1)%7 == 0 || i == len(chars)-1 {
			rows = append(rows, cur)
			cur = tele.Row{}
		}
	}
	m.Inline(rows...)
	return m
}
