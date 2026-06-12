package bot

import (
	"fmt"

	"strings"

	"GapGame/internal/game/word_guess"
	"GapGame/internal/service"

	"GapGame/internal/session"

	"GapGame/internal/user"
	"GapGame/internal/utils"
	"GapGame/pkg/messages"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (h *Handler) StartHandler(c tele.Context) error {

	payload := c.Message().Payload
	if strings.HasPrefix(payload, "join_") {
		return h.handleGameJoin(c, payload)
	}

	u, isNew, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrUserFetch)
	}

	if isNew && len(c.Args()) > 0 {

		referrerID := strings.TrimPrefix(c.Args()[0], "r_")
		referrer, err := h.users.HandleReferral(u, referrerID)
		if err == nil {
			if _, err := h.bot.Send(&tele.User{ID: referrer.TelegramID},
				messages.ReferalEnterUser); err != nil {
				h.log.Error("failed to send referral notification", zap.Error(err))
			}
		}
	}

	if !u.IsProfileComplete() {
		return h.askNextProfileQuestion(c, u)
	}

	h.redis.ClearUserState(u.TelegramID)

	return editOrSend(c, messages.Welcome, MainMenuKeyboard())

}

func (h *Handler) TextHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrUserFetch)
	}

	text := strings.TrimSpace(c.Text())

	if handled, err := h.handleAdminTextInput(c, text); handled {
		return err
	}

	// see user_ profile

	if strings.HasPrefix(text, "/user_") {
		ID := strings.TrimPrefix(text, "/user_")

		if ID == u.ID {
			return h.sendProfileMessage(c, u, u, true)
		}

		target, err := h.users.GetByID(ID)
		if err != nil {
			return editOrSend(c, messages.ErrUserFetch)
		}

		return h.ShowUserProfile(c, u, target)
	}
	state, _ := h.redis.GetUserState(u.TelegramID)

	switch state {

	case session.StateCompleteAge:

		return h.handleProfileCompletion(c, u, "age", text)
	case session.StateCompleteProvince:

		return h.handleProfileCompletion(c, u, "province", text)
	case session.StateOptionalName:

		return h.handleOptionalField(c, u, "name", text)
	case session.StateOptionalCity:

		return h.handleOptionalField(c, u, "city", text)
	case session.StateEditName:

		return h.HandleEditText(c, u, "name", text)
	case session.StateEditAge:

		return h.HandleEditText(c, u, "age", text)
	case session.StateEditGender:

		return h.HandleEditText(c, u, "gender", text)
	case session.StateEditCity:

		return h.HandleEditText(c, u, "city", text)
	case session.StateEditProvince:

		return h.HandleEditText(c, u, "province", text)
	case session.StateDM:

		return h.handleDMMessage(c, u, text)

	case session.StateAddContactLabel:
		return h.HandleAddContactLabel(c)
	}

	// Forward in active chat session or game room
	room := h.rooms.GetRoomByPlayerID(u.TelegramID)
	if room != nil && !room.GameOver {
		if game, ok := room.State.(*word_guess.GameWordGuess); ok {
			if handled, err := h.handleWordGameText(c, room, game, text); handled {
				return err
			}
		}
	}

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	cs, err := h.redis.GetActiveChat(ctx, u.TelegramID)
	if err == nil && cs != nil {
		// Forward message to partner
		return h.forwardTextToPartner(c, u, cs, text)
	}

	// If in game room but no active anonymous chat, forward message as chat to the friend
	if room != nil && room.Player2 != nil {
		opponentID := room.Player1.ID
		if u.TelegramID == room.Player1.ID {
			opponentID = room.Player2.ID
		}
		_, err := h.bot.Send(&tele.User{ID: opponentID}, text)
		return err
	}

	return editOrSend(c, messages.UseMenu, MainMenuKeyboard())

}

func (h *Handler) MediaHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return editOrSend(c, messages.ErrUserFetch)
	}

	state, _ := h.redis.GetUserState(u.TelegramID)

	switch state {
	case session.StateOptionalPhoto, session.StateEditPhoto:
		return h.HandleEditPhoto(c, u)
	}

	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	cs, err := h.redis.GetActiveChat(ctx, u.TelegramID)
	if err == nil && cs != nil {
		return h.forwardMediaToPartner(c, u, cs)
	}

	// If in play-with-friends game room, forward message as chat to the friend
	room := h.rooms.GetRoomByPlayerID(u.TelegramID)
	if room != nil && room.Player2 != nil {
		opponentID := room.Player1.ID
		if u.TelegramID == room.Player1.ID {
			opponentID = room.Player2.ID
		}
		_, err := h.bot.Copy(&tele.User{ID: opponentID}, c.Message())
		return err
	}

	return editOrSend(c, messages.NeedActiveChatFirst)
}

func (h *Handler) forwardMediaToPartner(c tele.Context, u *user.User, cs *session.ChatSession) error {
	partnerID := cs.User1ID
	if partnerID == u.TelegramID {
		partnerID = cs.User2ID
	}

	msg := c.Message()
	if msg == nil {
		return editOrSend(c, messages.ErrMessageNotFound)
	}

	recipient := &tele.User{
		ID: partnerID,
	}

	_, err := h.bot.Copy(recipient, msg)
	if err != nil {
		fmt.Println("bot.Copy error:", err)
		return editOrSend(c, messages.ErrSendToPartner)
	}

	return nil
}

func (h *Handler) LocationHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrUserFetch)
	}

	state, _ := h.redis.GetUserState(u.TelegramID)

	switch state {

	case session.StateOptionalGPS, session.StateEditGPS:

		return h.HandleEditGPS(c, u)

	}

	return c.Respond(&tele.CallbackResponse{
		Text: messages.SendYourLocation,
	})
}

func (h *Handler) HandleGenderCallback(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	gender := string(user.Female)

	if c.Callback().Unique == "gender_male" {

		gender = string(user.Male)
	}

	if err := h.users.CompleteProfileField(u, "gender", gender); err != nil {

		return editOrSend(c, messages.ErrSetGender, MainMenuKeyboard())
	}

	c.Respond()

	return h.askNextProfileQuestion(c, u)

}

func (h *Handler) askNextProfileQuestion(c tele.Context, u *user.User) error {
	switch u.ProfileState {

	case user.StateNeedGender:

		return editOrSend(c, fmt.Sprintf(messages.StartMessage, c.Sender().FirstName), GenderKeyboard())
	case user.StateNeedAge:
		h.redis.SetUserState(u.TelegramID, session.StateCompleteAge)
		// c.Bot().Send(c.Sender(), "لطفاً سن خود را وارد کنید (عدد بین 13 تا 100):")
		return editOrSend(c, messages.StartAge)
	case user.StateNeedProvince:

		h.redis.SetUserState(u.TelegramID, session.StateCompleteProvince)
		return editOrSend(c, messages.StartProvince, ProvinceKeyboard())
	case user.StateComplete:

		h.redis.ClearUserState(u.TelegramID)
		return h.showOptionalPrompt(c, u)
	}

	return nil

}

func (h *Handler) handleProfileCompletion(c tele.Context, u *user.User, field, value string) error {

	if err := h.users.CompleteProfileField(u, field, value); err != nil {

		switch err {
		case service.ErrInvalidAge:
			return editOrSend(c, messages.ErrInvalidAgeMsg, RestartKeyboard())
		case service.ErrInvalidProvince:
			return editOrSend(c, messages.ErrInvalidProvinceMsg, ProvinceKeyboard())
		default:
			return editOrSend(c, messages.ErrInvalidInput, RestartKeyboard())
		}
	}

	return h.askNextProfileQuestion(c, u)

}

func (h *Handler) showOptionalPrompt(c tele.Context, u *user.User) error {
	missing := u.GetMissingOptionalFields()

	if len(missing) == 0 {
		return nil //c.Send("✅ پروفایل شما کامل است!", MainMenuKeyboard())
	}

	missingStr := strings.Join(missing, " , ")
	msg := fmt.Sprintf(
		messages.OptionalPrompt,
		len(missing),
		missingStr,
	)

	editOrSend(c, messages.CompleteReg, MainMenuKeyboard())

	return editOrSend(c, msg, OptionalCompletionKeyboard())

}

func (h *Handler) StartOptionalHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	c.Respond()

	if err := h.users.StartOptionalProfile(u); err != nil {

		return editOrSend(c, messages.ErrGeneric)
	}

	return h.askNextOptionalField(c, u)

}

func (h *Handler) SkipOptionalHandler(c tele.Context) error {

	c.Respond()

	return editOrSend(c, messages.OptionalSkipped, MainMenuKeyboard())

}

func (h *Handler) askNextOptionalField(c tele.Context, u *user.User) error {

	missing := u.GetMissingOptionalFields()

	if len(missing) == 0 {

		h.redis.ClearUserState(u.TelegramID)
		return editOrSend(c, messages.OptionalDone, MainMenuKeyboard())
	}

	switch missing[0] {

	case "نام":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalName)
		return editOrSend(c, messages.OptionalAskName, CancelKeyboard())
	case "شهر":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalCity)
		return editOrSend(c, messages.OptionalAskCity, CancelKeyboard())
	case "عکس پروفایل":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalPhoto)
		return editOrSend(c, messages.OptionalAskPhoto, CancelKeyboard())
	case "موقعیت مکانی":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalGPS)

		editOrSend(c, messages.GpsMsg1)

		return editOrSend(c, messages.GpsMsg2, LocationKeyboard())
	}

	return nil

}

func (h *Handler) handleOptionalField(c tele.Context, u *user.User, field, value string) error {

	if err := h.users.UpdateOptionalField(u, field, value); err != nil {

		switch err {
		case service.ErrInvalidName:
			return editOrSend(c, messages.ErrNameInvalid, CancelKeyboard())
		case service.ErrInvalidCity:
			return editOrSend(c, messages.ErrCityInvalid, CancelKeyboard())
		default:
			return editOrSend(c, messages.ErrInvalidInput, CancelKeyboard())
		}
	}

	return h.askNextOptionalField(c, u)

}
