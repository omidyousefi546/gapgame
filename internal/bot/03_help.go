package bot

import (
	"GapGame/pkg/messages"

	tele "gopkg.in/telebot.v3"
)

func (h *Handler) HelpHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpMsg, tele.ModeHTML)

}
func (h *Handler) HelpChatHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpChat, tele.ModeHTML)

}
func (h *Handler) HelpCreditHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpCredit, tele.ModeHTML)

}
func (h *Handler) HelpGpsHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpGps, tele.ModeHTML)

}
func (h *Handler) HelpProfileHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpProfile, tele.ModeHTML)

}
func (h *Handler) HelpSendchatHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpSendchat, tele.ModeHTML)

}
func (h *Handler) HelpDirectHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpDirect, tele.ModeHTML)

}
func (h *Handler) HelpShortcutsHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpShortcuts, tele.ModeHTML)

}
func (h *Handler) HelpOnwHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpOnw, tele.ModeHTML)

}
func (h *Handler) HelpChwHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpChw, tele.ModeHTML)

}
func (h *Handler) HelpContactsHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpContacts, tele.ModeHTML)

}
func (h *Handler) HelpSearchHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpSearch, tele.ModeHTML)

}
func (h *Handler) HelpDeleteMessageHandler(c tele.Context) error {

	return editOrSend(c, messages.HelpDeleteMessage, tele.ModeHTML)

}
func (h *Handler) GhavaninHandler(c tele.Context) error {

	return editOrSend(c, messages.Ghavanin, tele.ModeHTML)

}
