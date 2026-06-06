// internal/bot/game_handlers.go
package bot

import (
	"GapGame/internal/game/dare_and_truth"
	"GapGame/internal/game/dooz4"
	"GapGame/internal/game/dooz_classic"
	"GapGame/internal/game/game_manager"
	"GapGame/internal/session"
	"GapGame/internal/utils"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (h *Handler) showGamesHandler(c tele.Context) error {
	return c.Send("بازی خود را انتخاب کنید", GameMenuKeyboard())
}

func (h *Handler) selectGameHandler(c tele.Context) error {
	gameSelected := c.Data()

	var state game_manager.GameState
	var gameType string

	switch gameSelected {
	case "game_dooz4_normal":
		state = &dooz4.GameDooz4{}
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
		return c.Respond(&tele.CallbackResponse{Text: "بازی نامعتبر"})
	}

	// اگه قبلاً توی اتاقی بود، حذفش کن
	h.rooms.RemoveRoomsByPlayerID(c.Sender().ID)

	room := h.rooms.CreateRoom(c.Sender(), state)

	// فعال کردن session state برای بازیکن اول
	if err := h.redis.SetUserState(c.Sender().ID, session.StateInGame); err != nil {
		return err
	}

	link := fmt.Sprintf(
		"[ورود به بازی 🔗](https://ble.ir/%s?start=join_%d_%v)",

		utils.BOT_USERNAME, c.Sender().ID, state.GameType(),
	)
	msgText := fmt.Sprintf(utils.CreateGame, gameType, link)

	msg, err := h.bot.Send(c.Sender(), msgText, &tele.SendOptions{ParseMode: tele.ModeMarkdown}, InGameMenuKeyboard())
	if err != nil {
		return err
	}
	room.MsgID1 = msg.ID
	room.MsgID2 = msg.ID
	return nil
}

func (h *Handler) finishGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send(utils.ReStartMessage, MainMenuKeyboard())
	}
	return c.Send(utils.ReConfirmFinish, ConfirmFinishGameMenuKeyboard())
}

func (h *Handler) declineFinishGame(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send(utils.ReStartMessage, MainMenuKeyboard())
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
		return c.Send(utils.ReStartMessage, MainMenuKeyboard())
	}

	var opponent *tele.User
	if c.Sender().ID == room.Player1.ID {
		opponent = room.Player2
	} else {
		opponent = room.Player1
	}

	// پاک کردن session state هر دو بازیکن
	h.redis.ClearUserState(c.Sender().ID)
	if opponent != nil {
		h.redis.ClearUserState(opponent.ID)
		if _, err := h.bot.Send(opponent, utils.GameEndedByOpponent, MainMenuKeyboard()); err != nil {
			h.log.Error("failed to notify opponent", zap.Error(err))
		}
	}

	h.rooms.RemoveRoomByRoomID(room.ID)
	return c.Send(utils.GameEndedByYou, MainMenuKeyboard())
}

func (h *Handler) repeatGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send(utils.GameCancelledOpponent, MainMenuKeyboard())
	}
	// A repeat requires both players to still be present.
	if room.Player2 == nil {
		return c.Send(utils.GameCancelledOpponent, MainMenuKeyboard())
	}

	if c.Sender().ID != room.Player1.ID {
		room.Player2 = room.Player1
		room.Player1 = c.Sender()
	}
	room.Turn = room.Player1.ID
	room.State.Reset()

	// Pick the right board keyboard for whatever game type was being played.
	var board *tele.ReplyMarkup
	switch g := room.State.(type) {
	case *dooz4.GameDooz4:
		board = boardDooz4Keyboard(&g.Board)
	case *dooz_classic.GameDoozClassic:
		board = boardDoozClassicKeyboard(&g.Board)
	case *dare_and_truth.GameDareTruth:
		board = boardDareAndTruthKeyboard()
	default:
		return c.Send(utils.GameCancelledOpponent, MainMenuKeyboard())
	}

	text := fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v (%v)", room.Player1.FirstName, "🔴")

	if _, err := h.bot.Send(room.Player1, utils.GameDooz4ReStarted, InGameMenuKeyboard()); err != nil {
		h.log.Error("failed to send game restart to player1", zap.Error(err))
	}
	if _, err := h.bot.Send(room.Player2, utils.GameDooz4ReStarted, InGameMenuKeyboard()); err != nil {
		h.log.Error("failed to send game restart to player2", zap.Error(err))
	}

	msg1, err := h.bot.Send(room.Player1, text, board)
	if err != nil {
		h.log.Error("failed to send board to player1", zap.Error(err))
	} else {
		room.MsgID1 = msg1.ID
	}
	msg2, err := h.bot.Send(room.Player2, text, board)
	if err != nil {
		h.log.Error("failed to send board to player2", zap.Error(err))
	} else {
		room.MsgID2 = msg2.ID
	}
	return nil
}

func (h *Handler) cancelGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send(utils.GameCancelledOpponent, MainMenuKeyboard())
	}
	h.redis.ClearUserState(c.Sender().ID)
	h.rooms.RemoveRoomByRoomID(room.ID)
	return c.Send(utils.GameCancelledYou, MainMenuKeyboard())
}

func (h *Handler) handleGameJoin(c tele.Context, payload string) error {
	parts := strings.Split(payload, "_")
	// join_<playerID>_<gameType>  →  parts[1], parts[2]
	if len(parts) < 3 {
		return c.Send("لینک نامعتبر ❌", MainMenuKeyboard())
	}

	creatorID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Send("لینک نامعتبر ❌", MainMenuKeyboard())
	}

	// نمیتونه با خودش بازی کنه
	if c.Sender().ID == creatorID {
		return c.Respond(&tele.CallbackResponse{Text: "نمیتونی با خودت بازی کنی!"})
	}

	room := h.rooms.GetRoomByPlayerID(creatorID)
	if room == nil {
		return c.Send("بازی پیدا نشد یا منقضی شده ❌", MainMenuKeyboard())
	}
	if room.Player2 != nil {
		return c.Send("بازی پر شده ❗", MainMenuKeyboard())
	}

	room, ok := h.rooms.JoinRoom(room.ID, c.Sender())
	if !ok {
		return c.Send("بازی پر شده ❗", MainMenuKeyboard())
	}

	// فعال کردن session برای بازیکن دوم
	h.redis.SetUserState(c.Sender().ID, session.StateInGame)

	var msg string
	var keyboard *tele.ReplyMarkup

	switch parts[2] {
	case "gameDooz4Gravity":
		game := room.State.(*dooz4.GameDooz4)
		msg = utils.GameDooz4Started
		keyboard = boardDooz4Keyboard(&game.Board)
	case "gameDoozClassic":
		game := room.State.(*dooz_classic.GameDoozClassic)
		msg = utils.GameDoozClassicStarted
		keyboard = boardDoozClassicKeyboard(&game.Board)
	case "gameDareAndTruth":
		msg = utils.GameDareAndTruthStarted
		keyboard = boardDareAndTruthKeyboard()
	}

	if _, err := h.bot.Send(room.Player1, msg, InGameMenuKeyboard()); err != nil {
		h.log.Error("failed to send game start to player1", zap.Error(err))
	}
	if _, err := h.bot.Send(room.Player2, msg, InGameMenuKeyboard()); err != nil {
		h.log.Error("failed to send game start to player2", zap.Error(err))
	}
	room.StartRoom(h.bot, keyboard)
	return nil
}

// /////////////
func (h *Handler) moveDooz4GravityHandler(c tele.Context) error {
	data := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send("/start")
	}

	xy := strings.Split(data, "-")
	x, _ := strconv.Atoi(strings.TrimSpace(xy[0]))
	y, _ := strconv.Atoi(xy[1])

	status := room.State.(*dooz4.GameDooz4).MakeMove(
		h.bot, c, boardDooz4Keyboard, boardDooz4KeyboardDisabled, room, c.Sender(), x, y,
	)
	if status {
		if _, err := h.bot.Send(room.Player1, utils.GameRepeat, AfterGameMenuKeyboard()); err != nil {
			h.log.Error("failed to send game repeat to player1", zap.Error(err))
		}
		if _, err := h.bot.Send(room.Player2, utils.GameRepeat, AfterGameMenuKeyboard()); err != nil {
			h.log.Error("failed to send game repeat to player2", zap.Error(err))
		}
	}
	return nil
}

func (h *Handler) moveDoozClassicHandler(c tele.Context) error {
	data := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send("/start")
	}

	xy := strings.Split(data, "-")
	x, _ := strconv.Atoi(strings.TrimSpace(xy[0]))
	y, _ := strconv.Atoi(xy[1])

	status := room.State.(*dooz_classic.GameDoozClassic).MakeMove(
		h.bot, c, boardDoozClassicKeyboard, boardDoozClassicKeyboardDisabled, room, c.Sender(), x, y,
	)
	if status {
		if _, err := h.bot.Send(room.Player1, utils.GameRepeat, AfterGameMenuKeyboard()); err != nil {
			h.log.Error("failed to send game repeat to player1", zap.Error(err))
		}
		if _, err := h.bot.Send(room.Player2, utils.GameRepeat, AfterGameMenuKeyboard()); err != nil {
			h.log.Error("failed to send game repeat to player2", zap.Error(err))
		}
	}
	return nil
}

func (h *Handler) moveDareAndTruthHandler(c tele.Context) error {
	data := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Send("/start")
	}
	room.State.(*dare_and_truth.GameDareTruth).MakeMove(
		h.bot, c, boardDareAndTruthKeyboard(), room, c.Sender(), data,
	)
	return nil
}
