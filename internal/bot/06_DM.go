package bot

import (
	"GapGame/pkg/messages"
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
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	requesterID, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrUserNotFound})
	}

	requester, _, err := h.users.GetOrCreate(requesterID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrUserNotFound})
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
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrCreateChat})
	}

	c.Respond()

	if _, err := h.bot.Send(&tele.User{ID: requester.TelegramID}, messages.ChatRequestAccepted, ActiveChatKeyboard()); err != nil {
		h.log.Error("failed to notify requester", zap.Error(err))
	}

	return editOrSend(c, messages.ChatStarted, ActiveChatKeyboard())
}

func (h *Handler) RejectChatHandler(c tele.Context) error {
	requesterID, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	requester, _, err := h.users.GetOrCreate(requesterID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	c.Respond()

	if _, err := h.bot.Send(&tele.User{ID: requester.TelegramID}, messages.ChatRequestRejected); err != nil {
		h.log.Error("failed to notify requester", zap.Error(err))
	}

	return editOrSend(c, messages.RequestRejected)
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
		return editOrSend(c, messages.ErrTryAgain, MainMenuKeyboard())
	}

	target, _, err := h.users.GetOrCreate(targetID)

	if err != nil {

		return editOrSend(c, messages.ErrUserNotFound, MainMenuKeyboard())
	}

	senderID := fmt.Sprintf("/user_%s", u.ID)

	if _, err := h.bot.Send(&tele.User{ID: target.TelegramID},
		fmt.Sprintf(messages.DMFrom, senderID, text)); err != nil {
		h.log.Error("failed to send DM to target", zap.Error(err))
	}
	h.redis.ClearUserState(u.TelegramID)

	h.redis.ClearDMTarget(u.TelegramID)

	return editOrSend(c, messages.DMSent, MainMenuKeyboard())

}
