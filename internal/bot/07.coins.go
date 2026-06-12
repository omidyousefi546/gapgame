package bot

import (
	"GapGame/pkg/messages"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) CoinsHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrInfoFetch)
	}

	if c.Callback() != nil {

		c.Respond()
	}

	msg := fmt.Sprintf(
		messages.FreeCoin,
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

	return editOrSend(c, msg, keyboard, tele.ModeHTML)

}

// handler برای دکمه‌های خرید
func (h *Handler) BuyCoinsHandler(c tele.Context) error {
	c.Respond(&tele.CallbackResponse{
		Text: messages.ConnectingPayment,
	})

	// اینجا باید به درگاه پرداخت وصل بشی
	// فعلاً فقط یه پیام می‌فرستیم
	return editOrSend(c, messages.PaymentSoon)

}
