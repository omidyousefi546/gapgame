package bot

import (
	"GapGame/internal/utils"
	"fmt"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (h *Handler) InviteHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return c.Send("❌ خطا در دریافت اطلاعات")
	}

	if c.Callback() != nil {

		c.Respond()
	}
	inviteLink := fmt.Sprintf("https://ble.ir/%s?start=r_%v", utils.BOT_USERNAME, u.ID)

	// پیام اول - بنر قابل فوروارد
	msg1 := fmt.Sprintf(
		utils.InviteMsg1,
		inviteLink,
	)

	// پیام دوم - اطلاعات دعوت
	msg2 := fmt.Sprintf(
		utils.InviteMsg2,
		u.InviteCount,
	)

	if err := c.Send(msg1, tele.ModeHTML); err != nil {
		h.log.Error("failed to send invite message 1", zap.Error(err))
		return err
	}

	return c.Send(msg2, tele.ModeHTML, MainMenuKeyboard())

}
