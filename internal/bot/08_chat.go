// internal/bot/chat_handler.go — بازنویسی کامل

package bot

import (
	"fmt"
	"time"

	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/internal/utils"
	"GapGame/pkg/messages"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	chatCostGender  = 2
	chatCostNearby  = 4
	minChatDuration = time.Minute
)

func (h *Handler) ConnectHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Send(messages.ErrGeneric)
	}
	c.Respond()

	// ✅ Check if user has active chat
	hasActiveChat, err := h.redis.HasActiveChat(u.TelegramID)
	if err != nil {
		h.log.Error("redis error checking active chat", zap.Error(err))
		return c.Send(messages.ErrSystem)
	}
	if hasActiveChat {
		return c.Send(messages.AlreadyInActive, ActiveChatKeyboard())
	}

	return editOrSend(c, messages.ConnectIntro, ConnectMenuKeyboard())
}

// costForFilter returns the coin cost of a queue filter. Centralised so the
// «Search Again» flow charges exactly what the original search charged.
func costForFilter(filter string) int {
	switch filter {
	case "male", "female":
		return chatCostGender
	case "nearby_male", "nearby_female":
		return chatCostNearby
	default:
		return 0
	}
}

func (h *Handler) ConnectTypeHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond()
	}
	data := c.Callback().Data
	c.Respond()

	// ✅ Check if user has active chat
	hasActiveChat, err := h.redis.HasActiveChat(u.TelegramID)
	if err != nil {
		h.log.Error("redis error checking active chat", zap.Error(err))
		return c.Send(messages.ErrSystem)
	}
	if hasActiveChat {
		return editOrSend(c, messages.AlreadyInActive)
	}

	switch data {
	case "nearby":
		hasGPS := u.Latitude != nil && u.Longitude != nil
		return editOrSend(c, messages.ConnectNearbyIntro, NearbyMenuKeyboard(hasGPS))
	case "random":
		return h.joinQueue(c, u, "random", 0)
	case "male", "female":
		if u.Coins < chatCostGender {
			return editOrSend(c, fmt.Sprintf(messages.NeedCoinsForGender, chatCostGender, u.Coins))
		}
		return h.joinQueue(c, u, data, chatCostGender)
	}
	return nil
}

func (h *Handler) NearbyTypeHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond()
	}
	data := c.Callback().Data
	c.Respond()

	switch data {
	case "nearby_near":
		if u.Latitude == nil {
			return editOrSend(c, messages.NeedGPSForNearby)
		}
		return h.joinQueue(c, u, "nearby", 0)
	case "nearby_all":
		return h.joinQueue(c, u, "random", 0)
	case "nearby_male", "nearby_female":
		if u.Coins < chatCostNearby {
			return editOrSend(c, fmt.Sprintf(messages.NeedCoinsForGender, chatCostNearby, u.Coins))
		}
		return h.joinQueue(c, u, data, chatCostNearby)
	}
	return nil
}

// SearchAgainHandler re-runs the matching queue search with the exact filter
// the user previously selected (carried in the callback data).
func (h *Handler) SearchAgainHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	c.Respond()

	filter := c.Callback().Data
	if filter == "" {
		filter = "random"
	}

	cost := costForFilter(filter)
	if cost > 0 && u.Coins < cost {
		return editOrSend(c, fmt.Sprintf(messages.NeedCoinsForGender, cost, u.Coins))
	}

	return h.joinQueue(c, u, filter, cost)
}

// joinQueue — کسر سکه + ورود به صف (worker مچ میکنه)
func (h *Handler) joinQueue(c tele.Context, u *user.User, filter string, cost int) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// ✅ مرحله 1: بررسی اینکه کاربر قبلاً در صف نیست
	inQueue, err := h.redis.IsInQueue(u.TelegramID)
	if err != nil {
		return c.Send(messages.ErrSystem)
	}
	if inQueue {
		return editOrSend(c, messages.AlreadyInQueue)
	}

	// ✅ مرحله 2: بررسی چت فعال موجود
	hasActiveChat, err := h.redis.HasActiveChat(u.TelegramID)
	if err != nil {
		return c.Send(messages.ErrSystem)
	}
	if hasActiveChat {
		return editOrSend(c, messages.AlreadyInActive2)
	}

	// Deduct coins (the service re-reads the user, so we report the message
	// using the in-memory snapshot which is accurate enough for the user).
	if cost > 0 {
		if err := h.users.DeductCoins(u.TelegramID, cost); err != nil {
			return editOrSend(c, fmt.Sprintf(messages.NotEnoughCoins, u.Coins, cost))
		}
	}

	// ✅ مرحله 3: نمایش پیام "در حال جستجو"
	// روی کلیک دکمه شیشه‌ای، همان پیام منو ویرایش می‌شود (بدون دکمه لغو —
	// جستجو بعد از ۲ دقیقه به‌صورت خودکار تمام می‌شود).
	var msg *tele.Message
	if c.Callback() != nil && c.Message() != nil {
		if edited, eerr := h.bot.Edit(c.Message(), messages.Searching); eerr == nil {
			msg = edited
		}
	}
	if msg == nil {
		msg, err = h.bot.Send(c.Recipient(), messages.Searching)
		if err != nil {
			h.log.Error("error sending search message", zap.Error(err))
		}
	}

	entry := &session.QueueEntry{
		TelegramID: u.TelegramID,
		Gender:     string(u.Gender),
		Filter:     filter,
		Cost:       cost,
		JoinedAt:   time.Now(),
		Lat:        u.Latitude,
		Lon:        u.Longitude,
	}
	if msg != nil {
		entry.MessageID = msg.ID
	}

	// ✅ مرحله 4: عضویت در صف + علامت‌گذاری
	if err := h.redis.EnqueueChat(ctx, entry); err != nil {
		// برگشت سکه اگه enqueue خطا داد
		if cost > 0 {
			h.users.AwardCoinsByTelegramID(u.TelegramID, cost, "enqueue_error")
		}
		return editOrSend(c, messages.QueueJoinError)
	}

	// ✅ مرحله 5: علامت‌گذاری کاربر به‌عنوان "در صف"
	if err := h.redis.JoinQueue(u.TelegramID); err != nil {
		return editOrSend(c, messages.QueueJoinError)
	}

	return nil
}

func (h *Handler) ViewChatProfileHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond()
	}
	c.Respond()

	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	cs, err := h.redis.GetActiveChat(ctx, u.TelegramID)
	if err != nil || cs == nil {
		return c.Send(messages.NoActiveChat)
	}

	partnerID := cs.User1ID
	if partnerID == u.TelegramID {
		partnerID = cs.User2ID
	}

	partner, err := h.users.GetByTelegramID(partnerID)
	if err != nil {
		return c.Send(messages.ErrProfileFetch)
	}

	h.bot.Send(&tele.User{ID: partnerID}, messages.ProfileViewedNotice)

	return h.sendProfileMessage(c, u, partner, false)
}

func (h *Handler) EndChatHandler(c tele.Context) error {
	c.Respond()
	// "پایان چت" is a reply-keyboard button, so a new message is expected here.
	return editOrSend(c, messages.ConfirmEndChat, ConfirmEndChatKeyboard())
}

func (h *Handler) ConfirmEndChatHandler(c tele.Context) error {
	u, err := h.users.GetByTelegramID(c.Sender().ID)
	if err != nil {
		return err
	}

	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// ✅ استفاده از GetActiveChat برای دریافت ChatSession
	cs, err := h.redis.GetActiveChat(ctx, u.TelegramID)
	if err != nil || cs == nil {
		return c.Edit(messages.NoActiveChatShort)
	}

	partnerID := cs.User2ID
	if cs.User2ID == u.TelegramID {
		partnerID = cs.User1ID
	}

	// ✅ حذف active chat برای هر دو کاربر
	if err := h.redis.DeleteActiveChat(ctx, u.TelegramID); err != nil {
		// Error already logged by redis manager
	}

	// ✅ Remove any active game rooms and clear user states for both players
	if r1 := h.rooms.GetRoomByPlayerID(u.TelegramID); r1 != nil {
		h.rooms.RemoveRoomByRoomID(r1.ID)
	}
	if r2 := h.rooms.GetRoomByPlayerID(partnerID); r2 != nil {
		h.rooms.RemoveRoomByRoomID(r2.ID)
	}
	h.redis.ClearUserState(u.TelegramID)
	h.redis.ClearUserState(partnerID)

	partnerRef := messages.ErrUserNotFound
	if partner, perr := h.users.GetByTelegramID(partnerID); perr == nil {
		partnerRef = userPublicRef(partner.ID)
	}
	meRef := userPublicRef(u.ID)

	// ✅ ویرایش پیام تایید به پیام پایان چت (به‌جای ارسال پیام جدید)
	editOrSend(c, fmt.Sprintf(messages.ChatEndedByYou, partnerRef))
	// کیبورد منوی اصلی فقط با پیام جدید قابل ارسال است.
	h.bot.Send(&tele.User{ID: u.TelegramID}, messages.BackToMenu, MainMenuKeyboard())

	// ✅ ارسال پیام به شریک چت
	_, err = h.bot.Send(&tele.User{ID: partnerID}, fmt.Sprintf(
		messages.ChatEndedByPartner, meRef,
	), MainMenuKeyboard())
	return err
}

func (h *Handler) CancelEndChatHandler(c tele.Context) error {
	c.Respond()
	return c.Delete()
}
