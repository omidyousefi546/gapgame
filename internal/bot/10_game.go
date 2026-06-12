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
	"time"
	"unicode"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) menuKeyboardFor(playerID int64) *tele.ReplyMarkup {
	if h.redis.HasActiveChatSilent(playerID) {
		return ActiveChatKeyboard()
	}
	return MainMenuKeyboard()
}

func (h *Handler) showGamesHandler(c tele.Context) error {
	return editOrSend(c, messages.PickGame, GameMenuKeyboard())
}

func (h *Handler) sendAfterGameMenu(room *game_manager.Room) {
	if room.Player1 != nil {
		h.bot.Send(room.Player1, "🏁 بازی به پایان رسید. چه کاری می‌خواهید انجام دهید؟", AfterGameMenuKeyboard())
	}
	if room.Player2 != nil {
		h.bot.Send(room.Player2, "🏁 بازی به پایان رسید. چه کاری می‌خواهید انجام دهید؟", AfterGameMenuKeyboard())
	}
}

func (h *Handler) gameConfigFromSelection(gameSelected string) (game_manager.GameState, string, bool) {
	var state game_manager.GameState
	var gameName string

	switch gameSelected {
	case "game_dooz4_normal":
		state = &dooz4.GameDooz4Normal{}
		gameName = "بازی دوز ۴ عادی"
	case "game_dooz4_gravity":
		state = &dooz4.GameDooz4{}
		gameName = "بازی دوز ۴ جاذبه"
	case "game_dooz_classic":
		state = &dooz_classic.GameDoozClassic{}
		gameName = "بازی دوز کلاسیک"
	case "game_dare_and_truth":
		state = &dare_and_truth.GameDareTruth{}
		gameName = "جرات حقیقت"
	case "game_rps":
		state = &rps.GameRPS{Round: 1}
		gameName = "سنگ کاغذ قیچی"
	case "game_word_guess":
		state = &word_guess.GameWordGuess{State: word_guess.StateChoosingType}
		gameName = "حدس کلمه"
	default:
		return nil, "", false
	}
	return state, gameName, true
}

func gameSelectionFromState(state game_manager.GameState) string {
	if state == nil {
		return ""
	}
	switch state.GameType() {
	case "gameDooz4Gravity":
		return "game_dooz4_gravity"
	case "gameDooz4Normal":
		return "game_dooz4_normal"
	case "gameDoozClassic":
		return "game_dooz_classic"
	case "gameDareAndTruth":
		return "game_dare_and_truth"
	case "gameRPS":
		return "game_rps"
	case "gameWordGuess":
		return "game_word_guess"
	default:
		return ""
	}
}

func gameBoardForState(state game_manager.GameState) *tele.ReplyMarkup {
	switch g := state.(type) {
	case *dooz4.GameDooz4:
		return boardDooz4Keyboard(&g.Board, "game_dooz4_gravity")
	case *dooz4.GameDooz4Normal:
		return boardDooz4Keyboard(&g.Board, "game_dooz4_normal")
	case *dooz_classic.GameDoozClassic:
		return boardDoozClassicKeyboard(&g.Board)
	case *dare_and_truth.GameDareTruth:
		return boardDareAndTruthKeyboard()
	case *rps.GameRPS:
		return boardRPSKeyboard()
	case *word_guess.GameWordGuess:
		return boardWordTypeKeyboard()
	default:
		return nil
	}
}

func startTextForState(room *game_manager.Room) string {
	switch room.State.(type) {
	case *rps.GameRPS:
		return "🔄 راند 1 شروع شد. انتخاب خود را بزنید 👇"
	case *word_guess.GameWordGuess:
		return "نوع بازی را انتخاب کنید 👇"
	case *dare_and_truth.GameDareTruth:
		return fmt.Sprintf("🎮 برای طرف مقابلت یک مورد را انتخاب کن \nنوبت %v ", room.NameFor(room.Player1.ID))
	default:
		return fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v (%v)", room.NameFor(room.Player1.ID), "🔴")
	}
}

func (h *Handler) markPlayWithFriendsGameOver(room *game_manager.Room) {
	if room == nil || room.Player1 == nil || h.redis.HasActiveChatSilent(room.Player1.ID) {
		return
	}
	room.GameOver = true
	room.PendingGameType = ""
	room.PendingGameRequestedBy = 0
	h.redis.ClearUserState(room.Player1.ID)
	if room.Player2 != nil {
		h.redis.ClearUserState(room.Player2.ID)
	}
	h.rooms.SaveRoom(room)
}

func (h *Handler) requestPlayWithFriendsGame(c tele.Context, room *game_manager.Room, gameSelected, gameName string) error {
	if room == nil || room.Player1 == nil || room.Player2 == nil || h.redis.HasActiveChatSilent(c.Sender().ID) {
		return editOrSend(c, messages.GameUnavailable, h.menuKeyboardFor(c.Sender().ID))
	}
	if !room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: "یک بازی هنوز در جریان است. ابتدا آن را تمام کنید."})
	}
	opponent := room.Player1
	if c.Sender().ID == room.Player1.ID {
		opponent = room.Player2
	}
	room.PendingGameType = gameSelected
	room.PendingGameRequestedBy = c.Sender().ID
	h.rooms.SaveRoom(room)

	requesterName := room.NameFor(c.Sender().ID)
	h.bot.Send(opponent, fmt.Sprintf("🎮 %s درخواست شروع بازی «%s» را داده است. آیا قبول می‌کنید؟", requesterName, gameName), PlayWithFriendsGameRequestKeyboard(gameSelected))
	return editOrSend(c, fmt.Sprintf("✅ درخواست بازی «%s» برای طرف مقابل ارسال شد. بازی فقط بعد از قبول او شروع می‌شود.", gameName))
}

func ensurePlayerFirst(room *game_manager.Room, firstID int64) {
	if room == nil || room.Player2 == nil || room.Player1 == nil || room.Player1.ID == firstID {
		return
	}
	room.Player1, room.Player2 = room.Player2, room.Player1
	room.Player1Name, room.Player2Name = room.Player2Name, room.Player1Name
}

func (h *Handler) startPlayWithFriendsGame(room *game_manager.Room, firstPlayerID int64, state game_manager.GameState, gameName string) {
	ensurePlayerFirst(room, firstPlayerID)
	room.State = state
	room.Reset()
	room.Turn = room.Player1.ID
	room.LastMove = time.Now()

	h.redis.SetUserState(room.Player1.ID, session.StateInGame)
	h.redis.SetUserState(room.Player2.ID, session.StateInGame)

	h.bot.Send(room.Player1, fmt.Sprintf("✅ بازی «%s» شروع شد!", gameName), InGameMenuKeyboard())
	h.bot.Send(room.Player2, fmt.Sprintf("✅ بازی «%s» شروع شد!", gameName), InGameMenuKeyboard())

	keyboard := gameBoardForState(room.State)
	text := startTextForState(room)
	msg1, _ := h.bot.Send(room.Player1, text, keyboard)
	if msg1 != nil {
		room.MsgID1 = msg1.ID
	}
	msg2, _ := h.bot.Send(room.Player2, text, keyboard)
	if msg2 != nil {
		room.MsgID2 = msg2.ID
	}
	h.rooms.SaveRoom(room)
}

func (h *Handler) selectGameHandler(c tele.Context) error {
	gameSelected := c.Data()
	state, gameType, ok := h.gameConfigFromSelection(gameSelected)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: messages.InvalidGame})
	}

	existingRoom := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if h.redis.HasActiveChatSilent(c.Sender().ID) {
		if existingRoom != nil {
			return c.Respond(&tele.CallbackResponse{Text: "شما یک بازی فعال دارید. لطفاً ابتدا آن را تمام کنید."})
		}
	}

	// Play with Friends: once both players are in the room, selecting another
	// game must be a request to the same friend, not a fresh share-link room.
	if existingRoom != nil && !h.redis.HasActiveChatSilent(c.Sender().ID) && existingRoom.Player2 != nil {
		return h.requestPlayWithFriendsGame(c, existingRoom, gameSelected, gameType)
	}

	// A not-yet-joined friend invite can safely be replaced with a new link.
	h.rooms.RemoveRoomsByPlayerID(c.Sender().ID)
	room := h.rooms.CreateRoom(c.Sender(), state, h.playerGameName(c.Sender().ID))

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
		return editOrSend(c, messages.ReStartMessage, h.menuKeyboardFor(c.Sender().ID))
	}
	return editOrSend(c, messages.ReConfirmFinish, ConfirmFinishGameMenuKeyboard())
}

func (h *Handler) declineFinishGame(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.ReStartMessage, h.menuKeyboardFor(c.Sender().ID))
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
		return editOrSend(c, messages.ReStartMessage, h.menuKeyboardFor(c.Sender().ID))
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
		h.bot.Send(opponent, messages.GameEndedByOpponent, h.menuKeyboardFor(opponent.ID))
	}

	h.rooms.RemoveRoomByRoomID(room.ID)
	return editOrSend(c, messages.GameEndedByYou, h.menuKeyboardFor(c.Sender().ID))
}

func (h *Handler) repeatGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.GameCancelledOpponent, h.menuKeyboardFor(c.Sender().ID))
	}
	if room.Player2 == nil {
		return editOrSend(c, messages.GameCancelledOpponent, h.menuKeyboardFor(c.Sender().ID))
	}
	if !h.redis.HasActiveChatSilent(c.Sender().ID) {
		gameSelected := gameSelectionFromState(room.State)
		_, gameName, ok := h.gameConfigFromSelection(gameSelected)
		if !ok {
			return editOrSend(c, messages.GameCancelledOpponent, h.menuKeyboardFor(c.Sender().ID))
		}
		return h.requestPlayWithFriendsGame(c, room, gameSelected, gameName)
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
	case *rps.GameRPS:
		board = boardRPSKeyboard()
	case *word_guess.GameWordGuess:
		board = boardWordTypeKeyboard()
	default:
		return editOrSend(c, messages.GameCancelledOpponent, h.menuKeyboardFor(c.Sender().ID))
	}

	var text string
	switch room.State.(type) {
	case *rps.GameRPS:
		text = "🔄 راند 1 شروع شد. انتخاب خود را بزنید 👇"
	case *word_guess.GameWordGuess:
		text = "نوع بازی را انتخاب کنید 👇"
	case *dare_and_truth.GameDareTruth:
		text = fmt.Sprintf("🎮 برای طرف مقابلت یک مورد را انتخاب کن \nنوبت %v ", room.NameFor(room.Player1.ID))
	default:
		text = fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v ", room.NameFor(room.Player1.ID))
	}

	h.bot.Send(room.Player1, messages.GameDooz4ReStarted, h.menuKeyboardFor(room.Player1.ID))
	h.bot.Send(room.Player2, messages.GameDooz4ReStarted, h.menuKeyboardFor(room.Player2.ID))

	msg1, _ := h.bot.Send(room.Player1, text, board)
	if msg1 != nil {
		room.MsgID1 = msg1.ID
	}
	msg2, _ := h.bot.Send(room.Player2, text, board)
	if msg2 != nil {
		room.MsgID2 = msg2.ID
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) cancelGameHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return editOrSend(c, messages.GameCancelledOpponent, h.menuKeyboardFor(c.Sender().ID))
	}
	h.redis.ClearUserState(room.Player1.ID)
	var opponent *tele.User
	if c.Sender().ID == room.Player1.ID {
		opponent = room.Player2
	} else {
		opponent = room.Player1
	}
	if opponent != nil {
		h.redis.ClearUserState(opponent.ID)
		h.bot.Send(opponent, messages.GameCancelledOpponent, h.menuKeyboardFor(opponent.ID))
	}
	h.rooms.RemoveRoomByRoomID(room.ID)
	return editOrSend(c, messages.GameCancelledYou, h.menuKeyboardFor(c.Sender().ID))
}

func (h *Handler) PlayWithFriendsGameAcceptCallback(c tele.Context) error {
	gameSelected := c.Callback().Data
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil || room.Player1 == nil || room.Player2 == nil || h.redis.HasActiveChatSilent(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameUnavailable})
	}
	if room.PendingGameType == "" || room.PendingGameRequestedBy == 0 || room.PendingGameRequestedBy == c.Sender().ID || room.PendingGameType != gameSelected {
		return c.Respond(&tele.CallbackResponse{Text: "این درخواست بازی دیگر معتبر نیست."})
	}
	state, gameName, ok := h.gameConfigFromSelection(gameSelected)
	if !ok {
		return c.Respond(&tele.CallbackResponse{Text: messages.InvalidGame})
	}
	if c.Message() != nil {
		h.bot.Edit(c.Message(), "✅ درخواست بازی پذیرفته شد.")
	}
	requesterID := room.PendingGameRequestedBy
	h.startPlayWithFriendsGame(room, requesterID, state, gameName)
	return nil
}

func (h *Handler) PlayWithFriendsGameRejectCallback(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil || h.redis.HasActiveChatSilent(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameUnavailable})
	}
	requesterID := room.PendingGameRequestedBy
	room.PendingGameType = ""
	room.PendingGameRequestedBy = 0
	h.rooms.SaveRoom(room)
	if c.Message() != nil {
		h.bot.Edit(c.Message(), "❌ درخواست بازی رد شد.", AfterGameMenuKeyboard())
	}
	if requesterID != 0 && requesterID != c.Sender().ID {
		h.bot.Send(&tele.User{ID: requesterID}, "❌ طرف مقابل درخواست شروع بازی را رد کرد.", AfterGameMenuKeyboard())
	}
	return nil
}

func (h *Handler) handleGameJoin(c tele.Context, payload string) error {
	if h.rooms.GetRoomByPlayerID(c.Sender().ID) != nil {
		return c.Respond(&tele.CallbackResponse{Text: "شما یک بازی فعال دارید. لطفاً ابتدا آن را تمام کنید."})
	}
	parts := strings.Split(payload, "_")
	if len(parts) < 3 {
		return editOrSend(c, messages.GameInviteLink, h.menuKeyboardFor(c.Sender().ID))
	}
	creatorID, _ := strconv.ParseInt(parts[1], 10, 64)
	if c.Sender().ID == creatorID {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameSelfPlay})
	}
	room := h.rooms.GetRoomByPlayerID(creatorID)
	if room == nil || room.Player2 != nil {
		return editOrSend(c, messages.GameUnavailable, h.menuKeyboardFor(c.Sender().ID))
	}
	room, _ = h.rooms.JoinRoom(room.ID, c.Sender(), h.playerGameName(c.Sender().ID))
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
	case "gameRPS":
		msg = chatGameStartMessage("rps")
		keyboard = boardRPSKeyboard()
	case "gameWordGuess":
		msg = chatGameStartMessage("word")
		keyboard = boardWordTypeKeyboard()
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
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz4.GameDooz4).MakeMove(h.bot, c, func(b *[7][7]int) *tele.ReplyMarkup { return boardDooz4Keyboard(b, "game_dooz4_gravity") }, boardDooz4KeyboardDisabled, room, c.Sender(), x, y) {
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.redis.ClearUserState(room.Player1.ID)
			if room.Player2 != nil {
				h.redis.ClearUserState(room.Player2.ID)
			}
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.sendAfterGameMenu(room)
		}
		return nil
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDooz4NormalHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz4.GameDooz4Normal).MakeMove(h.bot, c, func(b *[7][7]int) *tele.ReplyMarkup { return boardDooz4Keyboard(b, "game_dooz4_normal") }, boardDooz4KeyboardDisabled, room, c.Sender(), x, y) {
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.redis.ClearUserState(room.Player1.ID)
			if room.Player2 != nil {
				h.redis.ClearUserState(room.Player2.ID)
			}
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.sendAfterGameMenu(room)
		}
		return nil
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDoozClassicHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	xy := strings.Split(c.Callback().Data, "-")
	x, _ := strconv.Atoi(xy[0])
	y, _ := strconv.Atoi(xy[1])
	if room.State.(*dooz_classic.GameDoozClassic).MakeMove(h.bot, c, boardDoozClassicKeyboard, boardDoozClassicKeyboardDisabled, room, c.Sender(), x, y) {
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.redis.ClearUserState(room.Player1.ID)
			if room.Player2 != nil {
				h.redis.ClearUserState(room.Player2.ID)
			}
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.sendAfterGameMenu(room)
		}
		return nil
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) moveDareAndTruthHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game, ok := room.State.(*dare_and_truth.GameDareTruth)
	if !ok {
		// Stale button from a previous game; never touch another game's state.
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game.MakeMove(h.bot, c, boardDareAndTruthKeyboard(), room, c.Sender(), c.Callback().Data)
	h.rooms.SaveRoom(room)
	return nil
}

// endDareAndTruthHandler terminates a truth-or-dare game via the inline
// «پایان بازی» button. The game has no natural ending, so this is the only
// proper way to finish it: the room is removed, both players' game states are
// cleared and each player gets back the reply keyboard that matches their
// situation (active-chat keyboard inside a chat, main menu otherwise) so the
// conversation continues normally.
func (h *Handler) endDareAndTruthHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		// Stale button: strip the dead board, acknowledge and make sure the
		// user still has a usable reply keyboard.
		c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
		if c.Message() != nil {
			c.Edit(messages.GameAlreadyEnded)
		}
		return c.Send(messages.GameAlreadyEnded, h.menuKeyboardFor(c.Sender().ID))
	}
	if _, ok := room.State.(*dare_and_truth.GameDareTruth); !ok {
		// Safety: this button must never end any other game type.
		return c.Respond(&tele.CallbackResponse{Text: messages.InvalidGame})
	}

	var opponent *tele.User
	if room.Player1 != nil && c.Sender().ID == room.Player1.ID {
		opponent = room.Player2
	} else {
		opponent = room.Player1
	}

	if h.redis.HasActiveChatSilent(c.Sender().ID) {
		// 1. Remove all game state first so no further moves can race with the cleanup.
		h.rooms.RemoveRoomByRoomID(room.ID)
		h.redis.ClearUserState(c.Sender().ID)
		if opponent != nil {
			h.redis.ClearUserState(opponent.ID)
		}

		// 2. Detach the inline board from the opponent's game message so stale
		//    category buttons cannot be pressed afterwards.
		if opponent != nil {
			opponentMsgID := room.MsgID1
			if room.Player2 != nil && opponent.ID == room.Player2.ID {
				opponentMsgID = room.MsgID2
			}
			if opponentMsgID != 0 {
				h.bot.Edit(&tele.StoredMessage{
					MessageID: strconv.Itoa(opponentMsgID),
					ChatID:    opponent.ID,
				}, messages.GameDareTruthEndedByOpponent)
			}
			// 3. Restore the opponent's correct reply keyboard (chat controls if
			//    they are still in an anonymous chat, main menu otherwise).
			h.bot.Send(opponent, messages.GameDareTruthEndedByOpponent, h.menuKeyboardFor(opponent.ID))
		}

		// 4. Same for the player who pressed the button: replace the board message
		//    and hand back the proper reply keyboard so chatting continues normally.
		if c.Message() != nil {
			c.Edit(messages.GameDareTruthEndedByYou)
		}
		return c.Send(messages.GameDareTruthEndedByYou, h.menuKeyboardFor(c.Sender().ID))
	} else {
		// Play with friends mode:
		h.markPlayWithFriendsGameOver(room)
		if opponent != nil {
			opponentMsgID := room.MsgID1
			if room.Player2 != nil && opponent.ID == room.Player2.ID {
				opponentMsgID = room.MsgID2
			}
			if opponentMsgID != 0 {
				h.bot.Edit(&tele.StoredMessage{
					MessageID: strconv.Itoa(opponentMsgID),
					ChatID:    opponent.ID,
				}, messages.GameDareTruthEndedByOpponent)
			}
			h.bot.Send(opponent, messages.GameDareTruthEndedByOpponent, AfterGameMenuKeyboard())
		}

		if c.Message() != nil {
			c.Edit(messages.GameDareTruthEndedByYou)
		}
		return c.Send(messages.GameDareTruthEndedByYou, AfterGameMenuKeyboard())
	}
}

func (h *Handler) ChatGameHandler(c tele.Context) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, err := h.redis.GetActiveChat(ctx, c.Sender().ID)
	if err != nil || cs == nil {
		return editOrSend(c, messages.GameNotInChat)
	}
	partnerID := cs.User1ID
	if partnerID == c.Sender().ID {
		partnerID = cs.User2ID
	}

	if h.rooms.GetRoomByPlayerID(c.Sender().ID) != nil || h.rooms.GetRoomByPlayerID(partnerID) != nil {
		return editOrSend(c, "❌ یک بازی در جریان است. لطفاً ابتدا بازی فعلی را به پایان برسانید.")
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
	if h.rooms.GetRoomByPlayerID(c.Sender().ID) != nil || h.rooms.GetRoomByPlayerID(partnerID) != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ یک بازی در جریان است. لطفاً ابتدا بازی فعلی را به پایان برسانید."})
	}

	gameName := map[string]string{
		"rps":            "سنگ کاغذ قیچی",
		"word":           "حدس کلمه",
		"dooz4":          "دوز ۴ تایی",
		"dooz3":          "دوز کلاسیک",
		"dare_and_truth": "جرات حقیقت",
	}[gameType]
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

	if h.rooms.GetRoomByPlayerID(c.Sender().ID) != nil || h.rooms.GetRoomByPlayerID(partnerID) != nil {
		h.bot.Delete(c.Message())
		return c.Respond(&tele.CallbackResponse{Text: "❌ یک بازی قبلاً شروع شده است."})
	}

	var state game_manager.GameState
	var kb *tele.ReplyMarkup
	startMsg := chatGameStartMessage(gameType)
	switch gameType {
	case "rps":
		state = &rps.GameRPS{Round: 1}
		kb = boardRPSKeyboard()
	case "word":
		state = &word_guess.GameWordGuess{State: word_guess.StateChoosingType}
		kb = boardWordTypeKeyboard()
	case "dooz4":
		state = &dooz4.GameDooz4Normal{}
		kb = boardDooz4Keyboard(&[7][7]int{}, "game_dooz4_normal")
	case "dooz3":
		state = &dooz_classic.GameDoozClassic{}
		kb = boardDoozClassicKeyboard(&[3][3]int{})
	case "dare_and_truth":
		state = &dare_and_truth.GameDareTruth{}
		kb = boardDareAndTruthKeyboard()
	default:
		return c.Respond(&tele.CallbackResponse{Text: messages.InvalidGame})
	}

	h.rooms.RemoveRoomsByPlayerID(c.Sender().ID)
	h.rooms.RemoveRoomsByPlayerID(partnerID)
	room := h.rooms.CreateRoom(c.Sender(), state, h.playerGameName(c.Sender().ID))
	room, _ = h.rooms.JoinRoom(room.ID, &tele.User{ID: partnerID}, h.playerGameName(partnerID))

	h.redis.SetUserState(c.Sender().ID, session.StateInGame)
	h.redis.SetUserState(partnerID, session.StateInGame)

	// Clear request message from accept side
	h.bot.Edit(c.Message(), "✅ درخواست بازی پذیرفته شد!")

	// Send intro/rules with normal chat buttons
	h.bot.Send(c.Sender(), startMsg, ActiveChatKeyboard())
	h.bot.Send(&tele.User{ID: partnerID}, startMsg, ActiveChatKeyboard())

	var boardText string
	switch gameType {
	case "rps":
		boardText = "🔄 راند 1 شروع شد. انتخاب خود را بزنید 👇"
	case "word":
		boardText = "نوع بازی را انتخاب کنید 👇"
	case "dare_and_truth":
		boardText = fmt.Sprintf("🎮 برای طرف مقابلت یک مورد را انتخاب کن \nنوبت %v ", room.NameFor(room.Player1.ID))
	default:
		boardText = fmt.Sprintf("🎮 بازی شروع شد \nنوبت %v (%v)", room.NameFor(room.Player1.ID), "🔴")
	}

	msg1, _ := h.bot.Send(c.Sender(), boardText, kb)
	if msg1 != nil {
		room.MsgID1 = msg1.ID
	}
	msg2, _ := h.bot.Send(&tele.User{ID: partnerID}, boardText, kb)
	if msg2 != nil {
		room.MsgID2 = msg2.ID
	}

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
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game := room.State.(*rps.GameRPS)
	if game.Round == 0 {
		game.Round = 1
	}
	if c.Sender().ID == room.Player1.ID {
		if game.P1Move != "" {
			return c.Respond(&tele.CallbackResponse{Text: "حرکت این راند شما ثبت شده است."})
		}
		game.P1Move = move
	} else {
		if game.P2Move != "" {
			return c.Respond(&tele.CallbackResponse{Text: "حرکت این راند شما ثبت شده است."})
		}
		game.P2Move = move
	}

	if game.P1Move != "" && game.P2Move != "" {
		winner := determineRPSRoundWinner(game.P1Move, game.P2Move)
		if winner == 1 {
			game.P1Score++
		} else if winner == 2 {
			game.P2Score++
		}

		p1Ref := room.NameFor(room.Player1.ID)
		p2Ref := room.NameFor(room.Player2.ID)
		roundResult := rpsRoundResultText(winner, p1Ref, p2Ref)
		msg := fmt.Sprintf("🏁 نتیجه راند %d:\n%s: %s\n%s: %s\n%s\n\nامتیازها:\n%s: %d\n%s: %d",
			game.Round, p1Ref, rpsEmoji(game.P1Move), p2Ref, rpsEmoji(game.P2Move), roundResult, p1Ref, game.P1Score, p2Ref, game.P2Score)

		if game.P1Score >= 3 || game.P2Score >= 3 {
			winnerRef := p1Ref
			if game.P2Score >= 3 {
				winnerRef = p2Ref
			}
			msg += fmt.Sprintf("\n\n🏆 برنده نهایی: %s\nبازی با رسیدن به ۳ امتیاز تمام شد.", winnerRef)
			
			if h.redis.HasActiveChatSilent(room.Player1.ID) {
				h.bot.Send(room.Player1, msg, h.menuKeyboardFor(room.Player1.ID))
				if room.Player2 != nil {
					h.bot.Send(room.Player2, msg, h.menuKeyboardFor(room.Player2.ID))
					h.redis.ClearUserState(room.Player2.ID)
				}
				h.redis.ClearUserState(room.Player1.ID)
				h.rooms.RemoveRoomByRoomID(room.ID)
			} else {
				h.markPlayWithFriendsGameOver(room)
				h.bot.Send(room.Player1, msg, AfterGameMenuKeyboard())
				if room.Player2 != nil {
					h.bot.Send(room.Player2, msg, AfterGameMenuKeyboard())
				}
			}
			return nil
		}

		game.P1Move = ""
		game.P2Move = ""
		game.Round++
		msg += fmt.Sprintf("\n\n🔄 راند %d شروع شد. انتخاب بعدی را بزنید.", game.Round)
		h.rooms.SaveRoom(room)
		h.bot.Send(room.Player1, msg, boardRPSKeyboard())
		if room.Player2 != nil {
			h.bot.Send(room.Player2, msg, boardRPSKeyboard())
		}
		return nil
	}

	h.rooms.SaveRoom(room)
	return c.Edit(messages.GameMoveRegistered)
}

func determineRPSRoundWinner(p1, p2 string) int {
	if p1 == p2 {
		return 0
	}
	if map[string]string{"rock": "scissors", "paper": "rock", "scissors": "paper"}[p1] == p2 {
		return 1
	}
	return 2
}

func rpsRoundResultText(winner int, p1Ref, p2Ref string) string {
	switch winner {
	case 1:
		return fmt.Sprintf("🎉 %s این راند را برد!", p1Ref)
	case 2:
		return fmt.Sprintf("🎉 %s این راند را برد!", p2Ref)
	default:
		return "🤝 این راند مساوی شد."
	}
}

func rpsEmoji(m string) string {
	return map[string]string{"rock": "🪨", "paper": "🧻", "scissors": "✂️"}[m]
}

func boardWordTypeKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(m.Data("🇮🇷", "word_type", "fa"), m.Data("🇺🇸", "word_type", "en"), m.Data("🔢", "word_type", "num")))
	return m
}

func boardWordLengthKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("۳", "word_len", "3"), m.Data("۴", "word_len", "4"), m.Data("۵", "word_len", "5"), m.Data("۶", "word_len", "6")),
		m.Row(m.Data("۷", "word_len", "7"), m.Data("۸", "word_len", "8"), m.Data("۹", "word_len", "9"), m.Data("۱۰", "word_len", "10")),
	)
	return m
}

func (h *Handler) wordTypeHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game := room.State.(*word_guess.GameWordGuess)
	if game.State != word_guess.StateChoosingType {
		return c.Respond(&tele.CallbackResponse{Text: "نوع بازی قبلاً انتخاب شده است."})
	}
	game.Type = c.Callback().Data
	game.State = word_guess.StateChoosingLength
	game.CreatorID = c.Sender().ID

	prompt := fmt.Sprintf("%s\n\n📏 تعداد کاراکترها/ارقام رمز در این بازی را انتخاب کن:", wordGameTypeLabel(game.Type))
	h.bot.Edit(c.Message(), prompt, boardWordLengthKeyboard())

	partnerID := room.Player1.ID
	if partnerID == c.Sender().ID && room.Player2 != nil {
		partnerID = room.Player2.ID
	}
	if partnerID != c.Sender().ID {
		h.bot.Send(&tele.User{ID: partnerID}, fmt.Sprintf("⏳ حریف در حال انتخاب تعداد کاراکترهای بازی است...\n%s", wordGameTypeLabel(game.Type)))
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) wordLengthHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game := room.State.(*word_guess.GameWordGuess)
	if game.State != word_guess.StateChoosingLength {
		return c.Respond(&tele.CallbackResponse{Text: "طول بازی قبلاً انتخاب شده است."})
	}
	if c.Sender().ID != game.CreatorID {
		return c.Respond(&tele.CallbackResponse{Text: "تنها شروع‌کننده بازی می‌تواند طول را تعیین کند."})
	}

	length, _ := strconv.Atoi(c.Callback().Data)
	game.TargetLength = length
	game.State = word_guess.StateWaitingSecrets
	game.CurrentTurn = room.Player1.ID

	creatorName := room.NameFor(game.CreatorID)
	announcement := fmt.Sprintf("🎮 تنظیمات بازی توسط %s تعیین شد:\n\n%s\n\n🔐 لطفاً رمز خود را به صورت پیام متنی ارسال کنید. رمز باید دقیقاً %d %s باشد.", creatorName, wordGameSummary(game), game.TargetLength, wordGameUnit(game.Type))

	h.bot.Edit(c.Message(), announcement)
	partnerID := room.Player1.ID
	if partnerID == c.Sender().ID && room.Player2 != nil {
		partnerID = room.Player2.ID
	}
	if partnerID != c.Sender().ID {
		h.bot.Send(&tele.User{ID: partnerID}, announcement)
	}
	h.rooms.SaveRoom(room)
	return nil
}

func (h *Handler) wordGuessMoveHandler(c tele.Context) error {
	room := h.rooms.GetRoomByPlayerID(c.Sender().ID)
	if room == nil {
		return nil
	}
	if room.GameOver {
		return c.Respond(&tele.CallbackResponse{Text: messages.GameAlreadyEnded})
	}
	game := room.State.(*word_guess.GameWordGuess)
	char := normalizeSecretInput(game.Type, c.Callback().Data)
	if game.Type == "num" {
		return c.Respond(&tele.CallbackResponse{Text: "در حالت عددی، حدس را به صورت پیام متنی ارسال کنید."})
	}
	if game.State != word_guess.StatePlaying {
		return c.Respond(&tele.CallbackResponse{Text: "بازی هنوز آماده نیست."})
	}
	if c.Sender().ID != game.CurrentTurn {
		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("نوبت شما نیست! نوبت فعلی: %s", room.NameFor(game.CurrentTurn))})
	}
	if len([]rune(char)) != 1 || !isValidSecretForType(game.Type, char) {
		return c.Respond(&tele.CallbackResponse{Text: fmt.Sprintf("حرف نامعتبر برای %s", wordGameTypeLabel(game.Type))})
	}
	if alreadyWordGuessed(game, c.Sender().ID, char, room) {
		return c.Respond(&tele.CallbackResponse{Text: "این حرف قبلاً انتخاب شده است."})
	}

	found := applyWordLetterGuess(game, c.Sender().ID, char, room)
	if !found {
		c.Respond(&tele.CallbackResponse{Text: "این حرف در کلمه حریف نیست."})
	} else {
		c.Respond(&tele.CallbackResponse{Text: "درست بود!"})
	}

	if wordPlayerHasWon(game, c.Sender().ID, room) {
		winnerName := room.NameFor(c.Sender().ID)
		loser := room.Player1
		if c.Sender().ID == room.Player1.ID && room.Player2 != nil {
			loser = room.Player2
		}
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.bot.Send(c.Sender(), fmt.Sprintf("🎉 تبریک! کلمه حریف را کامل حدس زدی و برنده شدی.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), h.menuKeyboardFor(c.Sender().ID))
			if loser.ID != c.Sender().ID {
				h.bot.Send(loser, fmt.Sprintf("💀 حریف کلمه شما را کامل حدس زد.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), h.menuKeyboardFor(loser.ID))
				h.redis.ClearUserState(loser.ID)
			}
			h.redis.ClearUserState(c.Sender().ID)
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.bot.Send(c.Sender(), fmt.Sprintf("🎉 تبریک! کلمه حریف را کامل حدس زدی و برنده شدی.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), AfterGameMenuKeyboard())
			if loser.ID != c.Sender().ID {
				h.bot.Send(loser, fmt.Sprintf("💀 حریف کلمه شما را کامل حدس زد.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), AfterGameMenuKeyboard())
			}
		}
		return nil
	}

	switchWordTurn(game, room)
	h.rooms.SaveRoom(room)
	h.sendWordGameStatus(room, game)
	return nil
}

func boardWordGuessKeyboard(game *word_guess.GameWordGuess) *tele.ReplyMarkup {
	return boardWordGuessKeyboardFor(game, 0, nil)
}

func boardWordGuessKeyboardFor(game *word_guess.GameWordGuess, playerID int64, room *game_manager.Room) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	var chars []string
	if game.Type == "fa" {
		chars = []string{"آ", "ا", "ب", "پ", "ت", "ث", "ج", "چ", "ح", "خ", "د", "ذ", "ر", "ز", "ژ", "س", "ش", "ص", "ض", "ط", "ظ", "ع", "غ", "ف", "ق", "ک", "گ", "ل", "م", "ن", "و", "ه", "ی"}
	} else if game.Type == "en" {
		chars = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
	} else {
		chars = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	}

	wrong := game.WrongGuesses
	display := game.DisplayWord
	if room != nil && playerID != 0 {
		if playerID == room.Player1.ID {
			wrong = game.P1Wrong
			display = game.P1Display
		} else if room.Player2 != nil {
			wrong = game.P2Wrong
			display = game.P2Display
		}
	}

	var rows []tele.Row
	var cur tele.Row
	for i, ch := range chars {
		lbl := ch
		if containsString(wrong, ch) {
			lbl = "✖️"
		} else if displayContainsLetter(display, ch) {
			lbl = "✅ " + ch
		}
		cur = append(cur, m.Data(lbl, "word_guess", ch))
		if (i+1)%7 == 0 || i == len(chars)-1 {
			rows = append(rows, cur)
			cur = tele.Row{}
		}
	}
	m.Inline(rows...)
	return m
}

func chatGameStartMessage(gameType string) string {
	switch gameType {
	case "rps":
		return "🎮 سنگ، کاغذ، قیچی شروع شد!\n\n" +
			"روش بازی:\n" +
			"• هر دو بازیکن در هر راند یکی از سنگ، کاغذ یا قیچی را انتخاب می‌کنند.\n" +
			"• سنگ قیچی را می‌برد، قیچی کاغذ را می‌برد، کاغذ سنگ را می‌برد.\n" +
			"• بازی چند راندی است و اولین بازیکنی که به ۳ امتیاز برسد فوراً برنده نهایی می‌شود.\n\n" +
			"انتخاب راند اول را بزنید 👇"
	case "word":
		return "🎮 بازی حدس کلمه/عدد شروع شد!\n\n" +
			"روش بازی:\n" +
			"• ابتدا تعداد کاراکترها و نوع بازی انتخاب می‌شود.\n" +
			"• سپس هر دو بازیکن باید رمز خود را به صورت پیام متنی برای ربات بفرستند. رمزها برای حریف نمایش داده نمی‌شود.\n" +
			"• در حالت عددی، بازیکنان نوبتی حدس کامل می‌فرستند و سیستم رقم‌های درست در جای درست را اعلام می‌کند.\n" +
			"• در حالت کلمه، با کیبورد حروف بازی می‌کنید؛ حرف درست در جای خودش آشکار و حرف اشتباه علامت‌گذاری می‌شود.\n" +
			"• نوبت‌ها ادامه دارد تا یکی رمز حریف را کامل حدس بزند.\n\n" +
			"تنظیمات بازی 👇"
	case "dooz4":
		return messages.GameDooz4Started
	case "dooz3":
		return messages.GameDoozClassicStarted
	case "dare_and_truth":
		return messages.GameDareAndTruthStarted
	default:
		return "🎮 بازی شروع شد!"
	}
}

func wordSecretPrompt(gameType string) string {
	switch gameType {
	case "num":
		return "🔐 هر دو بازیکن باید عدد مخفی خود را ارسال کنند.\n\nقوانین:\n• عدد باید با طول انتخاب‌شده هماهنگ باشد.\n• بعد از ثبت هر دو عدد، نوبت‌ها شروع می‌شود.\n• در نوبت خود حدس کامل را به صورت پیام متنی بفرستید."
	case "fa":
		return "🔐 هر دو بازیکن باید کلمه فارسی مخفی خود را ارسال کنند.\n\nقوانین:\n• کلمه باید با طول انتخاب‌شده هماهنگ باشد.\n• بعد از ثبت هر دو کلمه، کیبورد حروف نمایش داده می‌شود.\n• در نوبت خود یک حرف را انتخاب کنید."
	case "en":
		return "🔐 هر دو بازیکن باید کلمه انگلیسی مخفی خود را ارسال کنند.\n\nقوانین:\n• کلمه باید با طول انتخاب‌شده هماهنگ باشد.\n• بعد از ثبت هر دو کلمه، کیبورد حروف نمایش داده می‌شود.\n• در نوبت خود یک حرف را انتخاب کنید."
	default:
		return "🔐 رمز مخفی خود را ارسال کنید."
	}
}

func (h *Handler) handleWordGameText(c tele.Context, room *game_manager.Room, game *word_guess.GameWordGuess, text string) (bool, error) {
	if game.State == word_guess.StateWaitingForWord {
		game.State = word_guess.StateWaitingSecrets
	}
	if game.State == word_guess.StateWaitingSecrets {
		return true, h.handleWordSecretInput(c, room, game, text)
	}
	if game.State == word_guess.StatePlaying {
		if game.Type == "num" {
			return true, h.handleNumberGuessInput(c, room, game, text)
		}
		if game.Type == "fa" || game.Type == "en" {
			return true, h.handleWordGuessTextInput(c, room, game, text)
		}
	}
	return false, nil
}

func (h *Handler) handleWordSecretInput(c tele.Context, room *game_manager.Room, game *word_guess.GameWordGuess, text string) error {
	secret := normalizeSecretInput(game.Type, text)
	if !isValidSecretForType(game.Type, secret) {
		return editOrSend(c, fmt.Sprintf("❌ ورودی نامعتبر است.\n%s\nرمز باید بدون فاصله و دقیقاً مطابق نوع بازی باشد.", wordGameSummary(game)))
	}
	if len([]rune(secret)) != game.TargetLength {
		return editOrSend(c, fmt.Sprintf("❌ طول رمز ارسالی اشتباه است!\n%s\nرمز شما باید دقیقاً %d %s باشد.", wordGameSummary(game), game.TargetLength, wordGameUnit(game.Type)))
	}

	if c.Sender().ID == room.Player1.ID {
		if game.P1Ready {
			return editOrSend(c, "✅ رمز شما قبلاً ثبت شده است. منتظر حریف بمانید.")
		}
		game.P1Secret = secret
		game.P1Ready = true
	} else {
		if game.P2Ready {
			return editOrSend(c, "✅ رمز شما قبلاً ثبت شده است. منتظر حریف بمانید.")
		}
		game.P2Secret = secret
		game.P2Ready = true
	}
	h.rooms.SaveRoom(room)
	h.bot.Send(c.Sender(), fmt.Sprintf("%s\n%s", messages.GameWordSet, wordGameSummary(game)))

	if game.P1Ready && game.P2Ready {
		game.State = word_guess.StatePlaying
		game.CurrentTurn = room.Player1.ID
		game.P1Display = hiddenDisplay(game.P2Secret)
		game.P2Display = hiddenDisplay(game.P1Secret)
		h.rooms.SaveRoom(room)
		h.sendWordGameStatus(room, game)
	} else {
		opponent := room.Player1
		if c.Sender().ID == room.Player1.ID && room.Player2 != nil {
			opponent = room.Player2
		}
		if opponent.ID != c.Sender().ID {
			h.bot.Send(opponent, fmt.Sprintf("✅ حریف رمز خود را ثبت کرد. منتظر ثبت رمز شما هستیم.\n%s", wordGameSummary(game)))
		}
	}
	return nil
}

func (h *Handler) handleNumberGuessInput(c tele.Context, room *game_manager.Room, game *word_guess.GameWordGuess, text string) error {
	if c.Sender().ID != game.CurrentTurn {
		return editOrSend(c, "⏳ نوبت شما نیست. منتظر حدس حریف باشید.")
	}
	guess := normalizeSecretInput("num", text)
	if !isValidSecretForType("num", guess) {
		return editOrSend(c, fmt.Sprintf("❌ ورودی نامعتبر است.\n%s\nفقط رقم بفرستید.", wordGameSummary(game)))
	}
	if len([]rune(guess)) != game.TargetLength {
		return editOrSend(c, fmt.Sprintf("❌ حدس شما باید دقیقاً %d %s باشد.\n%s", game.TargetLength, wordGameUnit(game.Type), wordGameSummary(game)))
	}

	target := game.P1Secret
	if c.Sender().ID == room.Player1.ID {
		target = game.P2Secret
	}

	result := numberGuessResult(target, guess)
	guessName := room.NameFor(c.Sender().ID)
	msg := fmt.Sprintf("%s\n🔢 حدس %s: %s\n%s", wordGameSummary(game), guessName, guess, result)
	if guess == target {
		msg += fmt.Sprintf("\n\n🏆 %s عدد حریف را کامل حدس زد و برنده شد!", guessName)
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.bot.Send(room.Player1, msg, h.menuKeyboardFor(room.Player1.ID))
			if room.Player2 != nil {
				h.bot.Send(room.Player2, msg, h.menuKeyboardFor(room.Player2.ID))
				h.redis.ClearUserState(room.Player2.ID)
			}
			h.redis.ClearUserState(room.Player1.ID)
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.bot.Send(room.Player1, msg, AfterGameMenuKeyboard())
			if room.Player2 != nil {
				h.bot.Send(room.Player2, msg, AfterGameMenuKeyboard())
			}
		}
		return nil
	}
	switchWordTurn(game, room)
	h.rooms.SaveRoom(room)
	msg += fmt.Sprintf("\n\nنوبت بعدی: %s", room.NameFor(game.CurrentTurn))
	h.bot.Send(room.Player1, msg)
	if room.Player2 != nil {
		h.bot.Send(room.Player2, msg)
	}
	return nil
}

func (h *Handler) sendWordGameStatus(room *game_manager.Room, game *word_guess.GameWordGuess) {
	p1Text := h.wordStatusForPlayer(room, game, room.Player1.ID)
	var p2Text string
	if room.Player2 != nil {
		p2Text = h.wordStatusForPlayer(room, game, room.Player2.ID)
	}
	if game.Type == "num" {
		if room.MsgID1 != 0 {
			h.bot.Edit(&tele.StoredMessage{MessageID: strconv.Itoa(room.MsgID1), ChatID: room.Player1.ID}, p1Text)
		} else if msg, err := h.bot.Send(room.Player1, p1Text); err == nil {
			room.MsgID1 = msg.ID
		}
		if room.Player2 != nil {
			if room.MsgID2 != 0 {
				h.bot.Edit(&tele.StoredMessage{MessageID: strconv.Itoa(room.MsgID2), ChatID: room.Player2.ID}, p2Text)
			} else if msg, err := h.bot.Send(room.Player2, p2Text); err == nil {
				room.MsgID2 = msg.ID
			}
		}
		h.rooms.SaveRoom(room)
		return
	}

	kb1 := boardWordGuessKeyboardFor(game, room.Player1.ID, room)
	var kb2 *tele.ReplyMarkup
	if room.Player2 != nil {
		kb2 = boardWordGuessKeyboardFor(game, room.Player2.ID, room)
	}
	if room.MsgID1 != 0 {
		h.bot.Edit(&tele.StoredMessage{MessageID: strconv.Itoa(room.MsgID1), ChatID: room.Player1.ID}, p1Text, kb1)
	} else if msg, err := h.bot.Send(room.Player1, p1Text, kb1); err == nil {
		room.MsgID1 = msg.ID
	}
	if room.Player2 != nil {
		if room.MsgID2 != 0 {
			h.bot.Edit(&tele.StoredMessage{MessageID: strconv.Itoa(room.MsgID2), ChatID: room.Player2.ID}, p2Text, kb2)
		} else if msg, err := h.bot.Send(room.Player2, p2Text, kb2); err == nil {
			room.MsgID2 = msg.ID
		}
	}
	h.rooms.SaveRoom(room)
}

func (h *Handler) wordStatusForPlayer(room *game_manager.Room, game *word_guess.GameWordGuess, playerID int64) string {
	if game.Type == "num" {
		return fmt.Sprintf(messages.GameWordReady, fmt.Sprintf("%s\n%s\n\nدر نوبت خود حدس کامل را به صورت پیام متنی ارسال کنید.", wordGameSummary(game), wordTurnLine(room, game)))
	}
	display := game.P1Display
	wrong := game.P1Wrong
	if room.Player2 != nil && playerID == room.Player2.ID {
		display = game.P2Display
		wrong = game.P2Wrong
	}
	wrongText := "ندارد"
	if len(wrong) > 0 {
		wrongText = strings.Join(wrong, "، ")
	}
	body := fmt.Sprintf("%s\nکلمه حریف: %s\nحروف اشتباه شما: %s\n%s\n\nاگر نوبت شماست، یک حرف انتخاب کنید یا فقط همان یک حرف را به صورت پیام متنی بفرستید.", wordGameSummary(game), display, wrongText, wordTurnLine(room, game))
	return fmt.Sprintf(messages.GameWordReady, body)
}

func (h *Handler) handleWordGuessTextInput(c tele.Context, room *game_manager.Room, game *word_guess.GameWordGuess, text string) error {
	if c.Sender().ID != game.CurrentTurn {
		return editOrSend(c, fmt.Sprintf("⏳ نوبت شما نیست. %s", wordTurnLine(room, game)))
	}
	char := normalizeSecretInput(game.Type, text)
	if len([]rune(char)) != 1 || !isValidSecretForType(game.Type, char) {
		return editOrSend(c, fmt.Sprintf("❌ حدس نامعتبر است.\n%s\nدر حالت کلمه باید دقیقاً یک حرف معتبر بفرستید یا از کیبورد حروف استفاده کنید.", wordGameSummary(game)))
	}
	if alreadyWordGuessed(game, c.Sender().ID, char, room) {
		return editOrSend(c, "این حرف قبلاً انتخاب شده است.")
	}

	found := applyWordLetterGuess(game, c.Sender().ID, char, room)
	resultText := "❌ این حرف در کلمه حریف نیست."
	if found {
		resultText = "✅ درست بود!"
	}

	if wordPlayerHasWon(game, c.Sender().ID, room) {
		winnerName := room.NameFor(c.Sender().ID)
		loser := room.Player1
		if c.Sender().ID == room.Player1.ID && room.Player2 != nil {
			loser = room.Player2
		}
		if h.redis.HasActiveChatSilent(room.Player1.ID) {
			h.bot.Send(c.Sender(), fmt.Sprintf("🎉 تبریک! کلمه حریف را کامل حدس زدی و برنده شدی.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), h.menuKeyboardFor(c.Sender().ID))
			if loser.ID != c.Sender().ID {
				h.bot.Send(loser, fmt.Sprintf("💀 حریف کلمه شما را کامل حدس زد.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), h.menuKeyboardFor(loser.ID))
				h.redis.ClearUserState(loser.ID)
			}
			h.redis.ClearUserState(c.Sender().ID)
			h.rooms.RemoveRoomByRoomID(room.ID)
		} else {
			h.markPlayWithFriendsGameOver(room)
			h.bot.Send(c.Sender(), fmt.Sprintf("🎉 تبریک! کلمه حریف را کامل حدس زدی و برنده شدی.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), AfterGameMenuKeyboard())
			if loser.ID != c.Sender().ID {
				h.bot.Send(loser, fmt.Sprintf("💀 حریف کلمه شما را کامل حدس زد.\n%s\nبرنده: %s", wordGameSummary(game), winnerName), AfterGameMenuKeyboard())
			}
		}
		return nil
	}

	switchWordTurn(game, room)
	h.rooms.SaveRoom(room)
	h.bot.Send(c.Sender(), resultText)
	h.sendWordGameStatus(room, game)
	return nil
}

func wordGameTypeLabel(gameType string) string {
	switch gameType {
	case "num":
		return "🔢 نوع بازی: عدد"
	case "fa":
		return "📝 نوع بازی: کلمه فارسی"
	case "en":
		return "📝 نوع بازی: کلمه انگلیسی"
	default:
		return "🎮 نوع بازی: نامشخص"
	}
}

func wordGameUnit(gameType string) string {
	if gameType == "num" {
		return "رقم"
	}
	return "کاراکتر"
}

func wordGameSummary(game *word_guess.GameWordGuess) string {
	return fmt.Sprintf("%s\n📏 تعداد لازم: %d %s", wordGameTypeLabel(game.Type), game.TargetLength, wordGameUnit(game.Type))
}

func wordTurnLine(room *game_manager.Room, game *word_guess.GameWordGuess) string {
	return fmt.Sprintf("نوبت فعلی: %s", room.NameFor(game.CurrentTurn))
}

func switchWordTurn(game *word_guess.GameWordGuess, room *game_manager.Room) {
	if game.CurrentTurn == room.Player1.ID && room.Player2 != nil {
		game.CurrentTurn = room.Player2.ID
	} else {
		game.CurrentTurn = room.Player1.ID
	}
}

func alreadyWordGuessed(game *word_guess.GameWordGuess, playerID int64, char string, room *game_manager.Room) bool {
	display := game.P1Display
	wrong := game.P1Wrong
	if room.Player2 != nil && playerID == room.Player2.ID {
		display = game.P2Display
		wrong = game.P2Wrong
	}
	return displayContainsLetter(display, char) || containsString(wrong, char)
}

func applyWordLetterGuess(game *word_guess.GameWordGuess, playerID int64, char string, room *game_manager.Room) bool {
	target := game.P2Secret
	display := game.P1Display
	wrong := game.P1Wrong
	isP1 := playerID == room.Player1.ID
	if !isP1 {
		target = game.P1Secret
		display = game.P2Display
		wrong = game.P2Wrong
	}

	found := false
	displayRunes := displaySlots(display)
	targetRunes := []rune(target)
	for i, r := range targetRunes {
		if string(r) == char {
			displayRunes[i] = string(r)
			found = true
		}
	}
	if !found {
		wrong = append(wrong, char)
	}
	newDisplay := strings.Join(displayRunes, " ")
	if isP1 {
		game.P1Display = newDisplay
		game.P1Wrong = wrong
	} else {
		game.P2Display = newDisplay
		game.P2Wrong = wrong
	}
	return found
}

func wordPlayerHasWon(game *word_guess.GameWordGuess, playerID int64, room *game_manager.Room) bool {
	if playerID == room.Player1.ID {
		return !strings.Contains(game.P1Display, "_")
	}
	return !strings.Contains(game.P2Display, "_")
}

func hiddenDisplay(secret string) string {
	parts := make([]string, 0, len([]rune(secret)))
	for range []rune(secret) {
		parts = append(parts, "_")
	}
	return strings.Join(parts, " ")
}

func displaySlots(display string) []string {
	parts := strings.Fields(display)
	if len(parts) == 0 {
		return []string{}
	}
	return parts
}

func displayContainsLetter(display, char string) bool {
	for _, part := range strings.Fields(display) {
		if part == char {
			return true
		}
	}
	return false
}

func normalizeSecretInput(gameType, text string) string {
	text = strings.TrimSpace(text)
	switch gameType {
	case "num":
		text = strings.NewReplacer(
			"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
			"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
		).Replace(text)
	case "en":
		text = strings.ToLower(text)
	case "fa":
		text = strings.NewReplacer("ي", "ی", "ى", "ی", "ك", "ک", "ة", "ه", "ۀ", "ه").Replace(text)
	}
	return text
}

func isValidSecretForType(gameType, text string) bool {
	l := len([]rune(text))
	if l < 1 || l > 12 || strings.ContainsAny(text, " \t\n\r") {
		return false
	}
	for _, r := range text {
		switch gameType {
		case "num":
			if r < '0' || r > '9' {
				return false
			}
		case "en":
			if r < 'a' || r > 'z' {
				return false
			}
		case "fa":
			if !unicode.IsLetter(r) || !((r >= 0x0600 && r <= 0x06FF) || (r >= 0x0750 && r <= 0x077F) || r == 'آ') {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func numberGuessResult(target, guess string) string {
	targetRunes := []rune(target)
	guessRunes := []rune(guess)
	var correct []string
	pattern := make([]string, len(targetRunes))
	for i := range pattern {
		pattern[i] = "_"
	}
	for i := 0; i < len(targetRunes) && i < len(guessRunes); i++ {
		if targetRunes[i] == guessRunes[i] {
			pattern[i] = string(guessRunes[i])
			correct = append(correct, fmt.Sprintf("%s (خانه %d)", string(guessRunes[i]), i+1))
		}
	}
	if len(correct) == 0 {
		return fmt.Sprintf("❌ هیچ رقمی در جای درست نیست.\nالگو: %s", strings.Join(pattern, " "))
	}
	return fmt.Sprintf("✅ رقم‌های درست در جای درست: %s\nالگو: %s", strings.Join(correct, "، "), strings.Join(pattern, " "))
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func (h *Handler) playerGameName(telegramID int64) string {
	if u, err := h.users.GetByTelegramID(telegramID); err == nil && u.Name != "" {
		return u.Name
	}
	return "کاربر"
}
