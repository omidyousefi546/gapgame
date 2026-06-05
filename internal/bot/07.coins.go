package bot

import (
	"GapGame/internal/utils"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) CoinsHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا در دریافت اطلاعات")
	}

	if c.Callback() != nil {

		c.Respond()
	}

	msg := fmt.Sprintf(
		utils.FreeCoin,
		u.Coins,
	)

	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(keyboard.Data("350 سکه - 249,000 تومان", "btnBuyCoins", "buy_350")),
		keyboard.Row(keyboard.Data("580 سکه - 389,000 تومان", "btnBuyCoins", "buy_580")),
		keyboard.Row(keyboard.Data("1700 سکه - 699,000 تومان", "btnBuyCoins", "buy_1700")),
		keyboard.Row(keyboard.Data("3500 سکه - 899,000 تومان", "btnBuyCoins", "buy_3500")),
		keyboard.Row(keyboard.Data("8500 سکه - 1,869,000 تومان", "btnBuyCoins", "buy_8500")),
		keyboard.Row(keyboard.Data("🎁 معرفی به دوستان (سکه رایگان)", "btnInviteFriends", "invite_friends")),
	)

	return c.Send(msg, keyboard, tele.ModeHTML)

}

// handler برای دکمه‌های خرید
func (h *Handler) BuyCoinsHandler(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{
		Text: "⏳ در حال اتصال به درگاه پرداخت...",
	})

	// اینجا باید به درگاه پرداخت وصل بشی
	// فعلاً فقط یه پیام می‌فرستیم
	return c.Send("🚧 سیستم پرداخت به زودی فعال می‌شود.")

}
