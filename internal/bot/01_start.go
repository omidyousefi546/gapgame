package bot

import (
	"fmt"

	"strings"

	"GapGame/internal/service"

	"GapGame/internal/session"

	"GapGame/internal/user"
	"GapGame/internal/utils"

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

		return c.Send("❌ خطا در دریافت اطلاعات کاربر")
	}

	if isNew && len(c.Args()) > 0 {

		referrerID := strings.TrimPrefix(c.Args()[0], "r_")
		referrer, err := h.users.HandleReferral(u, referrerID)
		if err == nil {
			if _, err := h.bot.Send(&tele.User{ID: referrer.TelegramID},
				utils.ReferalEnterUser); err != nil {
				h.log.Error("failed to send referral notification", zap.Error(err))
			}
		}
	}

	if !u.IsProfileComplete() {
		return h.askNextProfileQuestion(c, u)
	}

	h.redis.ClearUserState(u.TelegramID)

	return c.Send("به ربات چت ناشناس خوش آمدید! 👋", MainMenuKeyboard())

}

func (h *Handler) TextHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا در دریافت اطلاعات کاربر")
	}

	text := strings.TrimSpace(c.Text())

	// see user_ profile

	if strings.HasPrefix(text, "/user_") {
		ID := strings.TrimPrefix(text, "/user_")

		if ID == u.ID {
			return h.sendProfileMessage(c, u, u, true)
		}

		target, err := h.users.GetByID(ID)
		if err != nil {
			return c.Send("❌ خطا در دریافت اطلاعات کاربر")
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

	// Forward in active chat session
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	if cs, err := h.redis.GetActiveChat(ctx, u.TelegramID); err == nil && cs != nil {
		// بازی حدس کلمه - چک کردن اینکه آیا کاربر باید کلمه را تعیین کند
		room := h.rooms.GetRoomByPlayerID(u.TelegramID)
		if room != nil {
			if game, ok := room.State.(*word_guess.GameWordGuess); ok {
				if game.State == word_guess.StateWaitingForWord && game.CreatorID == u.TelegramID {
					word := strings.ToLower(strings.TrimSpace(text))
					if len([]rune(word)) < 3 || len([]rune(word)) > 6 {
						return c.Send("❌ طول کلمه باید بین ۳ تا ۶ کاراکتر باشد.")
					}

					game.TargetWord = word
					display := ""
					for range []rune(word) {
						display += "_ "
					}
					game.DisplayWord = strings.TrimSpace(display)
					game.State = word_guess.StatePlaying
					game.MaxTries = 6

					h.rooms.SaveRoom(room)

					h.bot.Send(c.Sender(), "✅ کلمه ثبت شد. بازی شروع شد!")

					msg := fmt.Sprintf("🎮 حریفت کلمه رو انتخاب کرد!\n\nکلمه: %s\n\nحروف یا اعداد رو حدس بزن:", game.DisplayWord)
					kb := boardWordGuessKeyboard(game)
					h.bot.Send(&tele.User{ID: game.GuesserID}, msg, kb)
					return nil
				}
			}
		}

		// Forward message to partner
		return h.forwardTextToPartner(c, u, cs, text)
	}

	return c.Send("لطفاً از منو استفاده کنید 👇", MainMenuKeyboard())

}

func (h *Handler) MediaHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا در دریافت اطلاعات کاربر")
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

	return c.Send("❌ ابتدا به یک چت فعال متصل شوید.")
}

func (h *Handler) forwardMediaToPartner(c tele.Context, u *user.User, cs *session.ChatSession) error {
	partnerID := cs.User1ID
	if partnerID == u.TelegramID {
		partnerID = cs.User2ID
	}

	msg := c.Message()
	if msg == nil {
		return c.Send("❌ پیام پیدا نشد.")
	}

	recipient := &tele.User{
		ID: partnerID,
	}

	_, err := h.bot.Copy(recipient, msg)
	if err != nil {
		fmt.Println("bot.Copy error:", err)
		return c.Send("❌ ارسال پیام به مخاطب با مشکل مواجه شد.")
	}

	return nil
}

func (h *Handler) LocationHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا در دریافت اطلاعات کاربر")
	}

	state, _ := h.redis.GetUserState(u.TelegramID)

	switch state {

	case session.StateOptionalGPS, session.StateEditGPS:

		return h.HandleEditGPS(c, u)

	}

	return c.Respond(&tele.CallbackResponse{
		Text: "لطفاً لوکیشن خود را بفرستید",
	})
}

func (h *Handler) HandleGenderCallback(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا", MainMenuKeyboard())
	}

	gender := string(user.Female)

	if c.Callback().Unique == "gender_male" {

		gender = string(user.Male)
	}

	if err := h.users.CompleteProfileField(u, "gender", gender); err != nil {

		return c.Send("❌ خطا در ثبت جنسیت", MainMenuKeyboard())
	}

	c.Respond()

	return h.askNextProfileQuestion(c, u)

}

func (h *Handler) askNextProfileQuestion(c tele.Context, u *user.User) error {
	switch u.ProfileState {

	case user.StateNeedGender:

		return c.Send(fmt.Sprintf(utils.StartMessage, c.Sender().FirstName), GenderKeyboard())
	case user.StateNeedAge:
		h.redis.SetUserState(u.TelegramID, session.StateCompleteAge)
		// c.Bot().Send(c.Sender(), "لطفاً سن خود را وارد کنید (عدد بین 13 تا 100):")
		return c.Send(utils.StartAge)
	case user.StateNeedProvince:

		h.redis.SetUserState(u.TelegramID, session.StateCompleteProvince)
		return c.Send(utils.StartProvince, ProvinceKeyboard())
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
			return c.Send("❌ سن باید عدد بین 13 تا 100 باشد", RestartKeyboard())
		case service.ErrInvalidProvince:
			return c.Send("❌ لطفاً نام استان خود را از منوی زیر انتخاب کنید", ProvinceKeyboard())
		default:
			return c.Send("❌ ورودی نامعتبر است", RestartKeyboard())
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
		"🔔 فقط %d قدم تا تکمیل پروفایل !\n\n"+
			"اطلاعات تکمیل نشده ی شما : %s\n\n"+
			"پروفایل خود را تکمیل کنید👇 و 5 سکه 💰 دریافت کنید.",
		len(missing),
		missingStr,
	)

	c.Send(utils.CompleteReg, MainMenuKeyboard())

	return c.Send(msg, OptionalCompletionKeyboard())

}

func (h *Handler) StartOptionalHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا", MainMenuKeyboard())
	}

	c.Respond()

	if err := h.users.StartOptionalProfile(u); err != nil {

		return c.Send("❌ خطا")
	}

	return h.askNextOptionalField(c, u)

}

func (h *Handler) SkipOptionalHandler(c tele.Context) error {

	c.Respond()

	return c.Send("باشه! هر وقت خواستی از منوی پروفایل تکمیل کن 😊", MainMenuKeyboard())

}

func (h *Handler) askNextOptionalField(c tele.Context, u *user.User) error {

	missing := u.GetMissingOptionalFields()

	if len(missing) == 0 {

		h.redis.ClearUserState(u.TelegramID)
		return c.Send("🎉 پروفایل کامل شد! 5 سکه دریافت کردید.", MainMenuKeyboard())
	}

	switch missing[0] {

	case "نام":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalName)
		return c.Send("نام خود را وارد کنید (حداکثر 20 کاراکتر فارسی):", CancelKeyboard())
	case "شهر":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalCity)
		return c.Send("شهر خود را به فارسی وارد کنید:", CancelKeyboard())
	case "عکس پروفایل":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalPhoto)
		return c.Send("عکس پروفایل خود را ارسال کنید:", CancelKeyboard())
	case "موقعیت مکانی":

		h.redis.SetUserState(u.TelegramID, session.StateOptionalGPS)

		c.Send(utils.GpsMsg1)

		return c.Send(utils.GpsMsg2, LocationKeyboard())
	}

	return nil

}

func (h *Handler) handleOptionalField(c tele.Context, u *user.User, field, value string) error {

	if err := h.users.UpdateOptionalField(u, field, value); err != nil {

		switch err {
		case service.ErrInvalidName:
			return c.Send("❌ نام باید فارسی و حداکثر 20 کاراکتر باشد", CancelKeyboard())
		case service.ErrInvalidCity:
			return c.Send("❌ نام شهر باید فارسی باشد", CancelKeyboard())
		default:
			return c.Send("❌ ورودی نامعتبر است", CancelKeyboard())
		}
	}

	return h.askNextOptionalField(c, u)

}
