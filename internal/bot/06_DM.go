package bot

import (
	"fmt"
	"strconv"
	"time"

	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/internal/utils"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (h *Handler) AcceptChatHandler(c tele.Context) error {
	target, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}

	requesterID, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ کاربر یافت نشد"})
	}

	requester, _, err := h.users.GetOrCreate(requesterID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ کاربر یافت نشد"})
	}

	// Create chat session
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	chatSession := &session.ChatSession{
		User1ID:   requester.TelegramID,
		User2ID:   target.TelegramID,
		StartedAt: time.Now(),
	}

	if err := h.redis.SetActiveChat(ctx, chatSession); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ایجاد چت"})
	}

	c.Respond()

	if _, err := h.bot.Send(&tele.User{ID: requester.TelegramID}, "✅ درخواست چت پذیرفته شد! شروع کنید 💬", ActiveChatKeyboard()); err != nil {
		h.log.Error("failed to notify requester", zap.Error(err))
	}

	return c.Send("✅ چت شروع شد! 💬", ActiveChatKeyboard())
}

func (h *Handler) RejectChatHandler(c tele.Context) error {
	requesterID, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}

	requester, _, err := h.users.GetOrCreate(requesterID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}

	c.Respond()

	if _, err := h.bot.Send(&tele.User{ID: requester.TelegramID}, "❌ درخواست چت رد شد."); err != nil {
		h.log.Error("failed to notify requester", zap.Error(err))
	}

	return c.Send("❌ درخواست رد شد.")
}

// forwardTextToPartner forwards a text message to the chat partner
func (h *Handler) forwardTextToPartner(c tele.Context, u *user.User, cs *session.ChatSession, text string) error {
	partnerID := cs.User2ID
	if cs.User2ID == u.TelegramID {
		partnerID = cs.User1ID
	}

	_, err := h.bot.Send(&tele.User{ID: partnerID}, text)
	return err
}

// DM
func (h *Handler) handleDMMessage(c tele.Context, u *user.User, text string) error {

	targetID, err := h.redis.GetDMTarget(u.TelegramID)

	if err != nil {

		h.redis.ClearUserState(u.TelegramID)
		return c.Send("❌ دوباره تلاش کنید", MainMenuKeyboard())
	}

	target, _, err := h.users.GetOrCreate(targetID)

	if err != nil {

		return c.Send("❌ کاربر یافت نشد", MainMenuKeyboard())
	}

	senderID := fmt.Sprintf("/user_%s", u.ID)

	if _, err := h.bot.Send(&tele.User{ID: target.TelegramID},
		"✉️ پیام از "+senderID+":\n\n"+text); err != nil {
		h.log.Error("failed to send DM to target", zap.Error(err))
	}
	h.redis.ClearUserState(u.TelegramID)

	h.redis.ClearDMTarget(u.TelegramID)

	return c.Send("✅ پیام ارسال شد.", MainMenuKeyboard())

}
