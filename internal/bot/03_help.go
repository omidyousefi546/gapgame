package bot

import (
	"GapGame/internal/utils"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) HelpHandler(c tele.Context) error {

	return c.Send(utils.HelpMsg, tele.ModeHTML)

}
func (h *Handler) HelpChatHandler(c tele.Context) error {

	return c.Send(utils.HelpChat, tele.ModeHTML)

}
func (h *Handler) HelpCreditHandler(c tele.Context) error {

	return c.Send(utils.HelpCredit, tele.ModeHTML)

}
func (h *Handler) HelpGpsHandler(c tele.Context) error {

	return c.Send(utils.HelpGps, tele.ModeHTML)

}
func (h *Handler) HelpProfileHandler(c tele.Context) error {

	return c.Send(utils.HelpProfile, tele.ModeHTML)

}
func (h *Handler) HelpSendchatHandler(c tele.Context) error {

	return c.Send(utils.HelpSendchat, tele.ModeHTML)

}
func (h *Handler) HelpDirectHandler(c tele.Context) error {

	return c.Send(utils.HelpDirect, tele.ModeHTML)

}
func (h *Handler) HelpShortcutsHandler(c tele.Context) error {

	return c.Send(utils.HelpShortcuts, tele.ModeHTML)

}
func (h *Handler) HelpOnwHandler(c tele.Context) error {

	return c.Send(utils.HelpOnw, tele.ModeHTML)

}
func (h *Handler) HelpChwHandler(c tele.Context) error {

	return c.Send(utils.HelpChw, tele.ModeHTML)

}
func (h *Handler) HelpContactsHandler(c tele.Context) error {

	return c.Send(utils.HelpContacts, tele.ModeHTML)

}
func (h *Handler) HelpSearchHandler(c tele.Context) error {

	return c.Send(utils.HelpSearch, tele.ModeHTML)

}
func (h *Handler) HelpDeleteMessageHandler(c tele.Context) error {

	return c.Send(utils.HelpDeleteMessage, tele.ModeHTML)

}
func (h *Handler) GhavaninHandler(c tele.Context) error {

	return c.Send(utils.Ghavanin, tele.ModeHTML)

}
