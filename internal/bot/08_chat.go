// internal/bot/chat_handler.go — بازنویسی کامل

package bot

import (
	"fmt"
	"time"

	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/internal/utils"

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
		return c.Send("❌ خطا")
	}
	c.Respond()

	// ✅ Check if user has active chat
	hasActiveChat, err := h.redis.HasActiveChat(u.TelegramID)
	if err != nil {
		h.log.Error("redis error checking active chat", zap.Error(err))
		return c.Send("❌ خطا در سیستم")
	}
	if hasActiveChat {
		return c.Send("شما الان در یک چت فعال هستید. ابتدا چت را پایان دهید.", ActiveChatKeyboard())
	}

	return c.Send("🔗 به یه ناشناس وصلم کن!\n\n👇 به کی وصلت کنم؟ انتخاب کن", ConnectMenuKeyboard())
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
		return c.Send("❌ خطا در سیستم")
	}
	if hasActiveChat {
		return c.Send("شما الان در یک چت فعال هستید.")
	}

	switch data {
	case "nearby":
		hasGPS := u.Latitude != nil && u.Longitude != nil
		return c.Edit(
			"🔗 به یه ناشناس وصلم کن!\n\n👇 چه کسی رو از افراد نزدیکت پیدا کنم؟",
			NearbyMenuKeyboard(hasGPS),
		)
	case "random":
		return h.joinQueue(c, u, "random", 0)
	case "male":
		if u.Coins < chatCostGender {
			return c.Send(fmt.Sprintf("❌ برای جستجوی پسر به %d سکه نیاز داری.\nسکه فعلی: %d", chatCostGender, u.Coins))
		}
		return h.joinQueue(c, u, "male", chatCostGender)
	case "female":
		if u.Coins < chatCostGender {
			return c.Send(fmt.Sprintf("❌ برای جستجوی دختر به %d سکه نیاز داری.\nسکه فعلی: %d", chatCostGender, u.Coins))
		}
		return h.joinQueue(c, u, "female", chatCostGender)
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
			return c.Send("❌ موقعیت مکانی ثبت نشده. ابتدا GPS رو در پروفایل ثبت کن.")
		}
		return h.joinQueue(c, u, "nearby", 0)
	case "nearby_all":
		return h.joinQueue(c, u, "random", 0)
	case "nearby_male":
		if u.Coins < chatCostNearby {
			return c.Send(fmt.Sprintf("❌ به %d سکه نیاز داری.\nسکه فعلی: %d", chatCostNearby, u.Coins))
		}
		return h.joinQueue(c, u, "nearby_male", chatCostNearby)
	case "nearby_female":
		if u.Coins < chatCostNearby {
			return c.Send(fmt.Sprintf("❌ به %d سکه نیاز داری.\nسکه فعلی: %d", chatCostNearby, u.Coins))
		}
		return h.joinQueue(c, u, "nearby_female", chatCostNearby)
	}
	return nil
}

// joinQueue — کسر سکه + ورود به صف (worker مچ میکنه)
func (h *Handler) joinQueue(c tele.Context, u *user.User, filter string, cost int) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// ✅ مرحله 1: بررسی اینکه کاربر قبلاً در صف نیست
	inQueue, err := h.redis.IsInQueue(u.TelegramID)
	if err != nil {
		return c.Send("❌ خطای سیستم")
	}
	if inQueue {
		return c.Send("⏳ شما قبلاً در صف هستید.\nبرای لغو می‌تونید دکمه‌ی پایین رو بزنید.")
	}

	// ✅ مرحله 2: بررسی چت فعال موجود
	hasActiveChat, err := h.redis.HasActiveChat(u.TelegramID)
	if err != nil {
		return c.Send("❌ خطای سیستم")
	}
	if hasActiveChat {
		return c.Send("🚫 شما الان در یک چت فعال هستید!\nابتدا چت را پایان دهید.")
	}

	// کسر سکه
	if cost > 0 {
		if err := h.users.DeductCoins(u.TelegramID, cost); err != nil {
			return c.Send(fmt.Sprintf(
				"❌ سکه کافی نداری!\nموجودی فعلی: %d سکه\nبرای این جستجو نیاز به %d سکه داری.",
				u.Coins, cost,
			))
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

	// ✅ مرحله 3: عضویت در صف + علامت‌گذاری
	if err := h.redis.EnqueueChat(ctx, entry); err != nil {
		// برگشت سکه اگه enqueue خطا داد
		if cost > 0 {
			h.users.AwardCoinsByTelegramID(u.TelegramID, cost, "enqueue_error")
		}
		return c.Send("❌ خطا در ورود به صف")
	}

	// ✅ مرحله 4: علامت‌گذاری کاربر به‌عنوان "در صف"
	if err := h.redis.JoinQueue(u.TelegramID); err != nil {
		return c.Send("❌ خطا در ورود به صف")
	}

	return c.Send("🔍 در حال جستجو...\nمنتظر بمون تا یه نفر پیدا بشه!", WaitingKeyboard())
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
		return c.Send("❌ چت فعالی وجود ندارد")
	}

	partnerID := cs.User1ID
	if partnerID == u.TelegramID {
		partnerID = cs.User2ID
	}

	partner, err := h.users.GetByTelegramID(partnerID)
	if err != nil {
		return c.Send("❌ خطا در دریافت پروفایل")
	}

	sysMsg := "🤖 پیام سیستم 👇\n\nمخاطب شما 《 پروفایل هاید چت 》 شما را مشاهده کرد.\n\n⚠️ توجه: پروفایل هاید چت اطلاعاتی است که در بخش پروفایل ربات ثبت کرده‌اید!"
	h.bot.Send(&tele.User{ID: partnerID}, sysMsg)

	return h.sendProfileMessage(c, u, partner, false)
}

func (h *Handler) EndChatHandler(c tele.Context) error {
	c.Respond()
	return c.Send("🤖 پیام سیستم 👇\n\nمطمئنی میخوای چت رو قطع کنی؟", ConfirmEndChatKeyboard())
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
		return c.Edit("❌ چتی فعال نیست.")
	}

	partnerID := cs.User2ID
	if cs.User2ID == u.TelegramID {
		partnerID = cs.User1ID
	}

	// ✅ حذف active chat برای هر دو کاربر
	if err := h.redis.DeleteActiveChat(ctx, u.TelegramID); err != nil {
		// Error already logged by redis manager
	}

	// ✅ ارسال پیام به کاربر شروع‌کننده
	h.bot.Send(&tele.User{ID: u.TelegramID}, fmt.Sprintf(
		"🎌 چت شما با /user_%d توسط شما پایان یافت.\nمیتونی با زدن '🚫 گزارش کاربر' در پروفایلش، تخلف رو گزارش بدی (/ghavanin).",
		partnerID,
	), MainMenuKeyboard())

	// ✅ ارسال پیام به شریک چت
	h.bot.Send(&tele.User{ID: partnerID}, fmt.Sprintf(
		"🎌 چت شما با /user_%d توسط مخاطبت پایان یافت.\nمیتونی با زدن '🚫 گزارش کاربر' در پروفایلش، تخلف رو گزارش بدی (/ghavanin).",
		u.TelegramID,
	), MainMenuKeyboard())

	// ✅ حذف دکمه callback
	return c.Delete()
}

func (h *Handler) CancelEndChatHandler(c tele.Context) error {
	c.Respond()
	return c.Delete()
}

func (h *Handler) CancelQueueHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond()
	}
	c.Respond()

	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// ✅ استفاده از روش جدید IsInQueue و LeaveQueue
	inQueue, err := h.redis.IsInQueue(u.TelegramID)
	if err != nil {
		h.log.Error("redis error checking queue", zap.Error(err))
		return c.Send("❌ خطا در سیستم")
	}
	if inQueue {
		// برگشت سکه اگه توی صف بوده
		filter, err := h.redis.GetWaitingFilter(ctx, u.TelegramID)
		if err != nil {
			h.log.Error("redis error getting filter", zap.Error(err))
		}
		if filter != "" {
			items, err := h.redis.GetQueueEntry(ctx, u.TelegramID, filter)
			if err != nil {
				h.log.Error("redis error getting queue entry", zap.Error(err))
			} else if items != nil && items.Cost > 0 {
				h.users.AwardCoinsByTelegramID(u.TelegramID, items.Cost, "cancel_queue")
				c.Send(fmt.Sprintf("💰 %d سکه‌ات برگشت داده شد.", items.Cost))
			}
			if err := h.redis.RemoveFromQueue(ctx, u.TelegramID, filter); err != nil {
				h.log.Error("redis error removing from queue", zap.Error(err))
			}
		}
		// ✅ حذف علامت "در صف" بودن
		if err := h.redis.LeaveQueue(u.TelegramID); err != nil {
			h.log.Error("redis error leaving queue", zap.Error(err))
		}
	}

	return c.Send("❌ از صف خارج شدید.", MainMenuKeyboard())
}
