package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"GapGame/pkg/messages"
	"GapGame/pkg/middleware"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// broadcastDelay throttles the broadcast loop so we stay well under the Bot
// API limit (~30 messages/second).
const broadcastDelay = 40 * time.Millisecond

// poorCoinThreshold: «give coins only to users with less than 2 coins».
const poorCoinThreshold = 2

type AdminPendingAction struct {
	Command              string
	Payload              string
	AwaitingConfirmation bool
}

const (
	adminCmdBroadcast     = "broadcast"
	adminCmdGiveCoinsAll  = "give_coins_all"
	adminCmdGiveCoinsPoor = "give_coins_poor"
	adminCmdBan           = "ban"
	adminCmdUnban         = "unban"
)

// registerAdminHandlers wires up every admin-only command behind the
// RequireAdmin middleware. Admin IDs come from the ADMIN_IDS env var.
func (h *Handler) registerAdminHandlers() {
	admin := middleware.RequireAdmin(&h.admins)

	h.bot.Handle("/admin", h.AdminHelpHandler, admin)
	h.bot.Handle("/broadcast", h.AdminBroadcastHandler, admin)
	h.bot.Handle("/give_coins_all", h.AdminGiveCoinsAllHandler, admin)
	h.bot.Handle("/give_coins_poor", h.AdminGiveCoinsPoorHandler, admin)
	h.bot.Handle("/ban", h.AdminBanHandler, admin)
	h.bot.Handle("/unban", h.AdminUnbanHandler, admin)
	h.bot.Handle("\fadmin_cmd", h.AdminCommandCallback, admin)
	h.bot.Handle("\fadmin_confirm", h.AdminConfirmCallback, admin)
	h.bot.Handle("\fadmin_cancel", h.AdminCancelCallback, admin)
}

// banGuardMiddleware blocks every interaction from banned users.
func (h *Handler) banGuardMiddleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil || h.admins[sender.ID] {
				return next(c)
			}
			banned, err := h.users.IsBanned(sender.ID)
			if err == nil && banned {
				return c.Send(messages.BannedNotice)
			}
			return next(c)
		}
	}
}

// AdminHelpHandler lists the available admin commands with inline actions.
func (h *Handler) AdminHelpHandler(c tele.Context) error {
	return c.Send(messages.AdminHelp, AdminMenuKeyboard())
}

func AdminMenuKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("📣 ارسال همگانی", "admin_cmd", adminCmdBroadcast)),
		m.Row(m.Data("💰 سکه به همه", "admin_cmd", adminCmdGiveCoinsAll)),
		m.Row(m.Data("🪙 سکه به کم‌سکه‌ها", "admin_cmd", adminCmdGiveCoinsPoor)),
		m.Row(m.Data("🚫 مسدود کردن", "admin_cmd", adminCmdBan), m.Data("✅ رفع مسدودی", "admin_cmd", adminCmdUnban)),
	)
	return m
}

func AdminCancelKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(m.Row(m.Data("❌ لغو عملیات", "admin_cancel", "cancel")))
	return m
}

func AdminConfirmKeyboard() *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("✅ تایید و اجرا", "admin_confirm", "yes")),
		m.Row(m.Data("❌ لغو", "admin_cancel", "cancel")),
	)
	return m
}

func (h *Handler) AdminCommandCallback(c tele.Context) error {
	c.Respond()
	return h.beginAdminAction(c, c.Callback().Data, "")
}

func (h *Handler) AdminCancelCallback(c tele.Context) error {
	c.Respond()
	h.clearAdminPending(c.Sender().ID)
	return editOrSend(c, messages.AdminCancelled)
}

func (h *Handler) AdminConfirmCallback(c tele.Context) error {
	c.Respond()
	pending := h.getAdminPending(c.Sender().ID)
	if pending == nil || !pending.AwaitingConfirmation {
		return editOrSend(c, messages.AdminNoPendingOperation, AdminMenuKeyboard())
	}
	h.clearAdminPending(c.Sender().ID)
	return h.executeAdminAction(c, pending)
}

// AdminBroadcastHandler starts the safe broadcast flow.
// Usage still works as /broadcast <text>, but execution waits for confirmation.
func (h *Handler) AdminBroadcastHandler(c tele.Context) error {
	return h.beginAdminAction(c, adminCmdBroadcast, strings.TrimSpace(c.Message().Payload))
}

// AdminGiveCoinsAllHandler starts the safe give-coins-to-all flow.
func (h *Handler) AdminGiveCoinsAllHandler(c tele.Context) error {
	return h.beginAdminAction(c, adminCmdGiveCoinsAll, strings.TrimSpace(c.Message().Payload))
}

// AdminGiveCoinsPoorHandler starts the safe give-coins-to-poor flow.
func (h *Handler) AdminGiveCoinsPoorHandler(c tele.Context) error {
	return h.beginAdminAction(c, adminCmdGiveCoinsPoor, strings.TrimSpace(c.Message().Payload))
}

// AdminBanHandler starts the safe ban flow. User references are public bot IDs
// only (/user_xxx), never Telegram IDs.
func (h *Handler) AdminBanHandler(c tele.Context) error {
	return h.beginAdminAction(c, adminCmdBan, strings.TrimSpace(c.Message().Payload))
}

// AdminUnbanHandler starts the safe unban flow. User references are public bot
// IDs only (/user_xxx), never Telegram IDs.
func (h *Handler) AdminUnbanHandler(c tele.Context) error {
	return h.beginAdminAction(c, adminCmdUnban, strings.TrimSpace(c.Message().Payload))
}

func (h *Handler) beginAdminAction(c tele.Context, command, payload string) error {
	if !isValidAdminCommand(command) {
		return editOrSend(c, messages.ErrInvalidCommand, AdminMenuKeyboard())
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		h.setAdminPending(c.Sender().ID, &AdminPendingAction{Command: command})
		return editOrSend(c, adminInputPrompt(command), AdminCancelKeyboard())
	}
	if errMsg := h.validateAdminPayload(command, payload); errMsg != "" {
		return editOrSend(c, errMsg, AdminCancelKeyboard())
	}
	pending := &AdminPendingAction{Command: command, Payload: payload, AwaitingConfirmation: true}
	h.setAdminPending(c.Sender().ID, pending)
	return editOrSend(c, fmt.Sprintf(messages.AdminConfirmPrompt, h.adminActionSummary(pending)), AdminConfirmKeyboard())
}

func (h *Handler) handleAdminTextInput(c tele.Context, text string) (bool, error) {
	if c.Sender() == nil || !h.admins[c.Sender().ID] {
		return false, nil
	}
	pending := h.getAdminPending(c.Sender().ID)
	if pending == nil {
		return false, nil
	}
	if pending.AwaitingConfirmation {
		return true, c.Send(messages.AdminSendInputFirst, AdminConfirmKeyboard())
	}

	text = strings.TrimSpace(text)
	if errMsg := h.validateAdminPayload(pending.Command, text); errMsg != "" {
		return true, c.Send(errMsg, AdminCancelKeyboard())
	}
	pending.Payload = text
	pending.AwaitingConfirmation = true
	h.setAdminPending(c.Sender().ID, pending)
	return true, c.Send(fmt.Sprintf(messages.AdminConfirmPrompt, h.adminActionSummary(pending)), AdminConfirmKeyboard())
}

func (h *Handler) executeAdminAction(c tele.Context, pending *AdminPendingAction) error {
	switch pending.Command {
	case adminCmdBroadcast:
		return h.executeAdminBroadcast(c, pending.Payload)
	case adminCmdGiveCoinsAll:
		amount, _ := parsePositiveAmount(pending.Payload)
		return h.executeAdminGiveCoinsAll(c, amount)
	case adminCmdGiveCoinsPoor:
		amount, _ := parsePositiveAmount(pending.Payload)
		return h.executeAdminGiveCoinsPoor(c, amount)
	case adminCmdBan:
		return h.setBanned(c, pending.Payload, true, messages.AdminUserBanned, messages.AdminBanNotifyUser)
	case adminCmdUnban:
		return h.setBanned(c, pending.Payload, false, messages.AdminUserUnbanned, messages.AdminUnbanNotifyUser)
	default:
		return c.Send(messages.ErrInvalidCommand)
	}
}

func (h *Handler) executeAdminBroadcast(c tele.Context, text string) error {
	ids, err := h.users.GetAllTelegramIDs()
	if err != nil {
		h.log.Error("broadcast: failed to load users", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}

	c.Send(fmt.Sprintf(messages.AdminBroadcastStarted, len(ids)))

	go func(adminID int64) {
		var ok, failed int
		for _, id := range ids {
			if _, err := h.bot.Send(&tele.User{ID: id}, text); err != nil {
				failed++
			} else {
				ok++
			}
			time.Sleep(broadcastDelay)
		}
		h.bot.Send(&tele.User{ID: adminID}, fmt.Sprintf(messages.AdminBroadcastFinished, ok, failed))
		h.log.Info("broadcast finished", zap.Int("ok", ok), zap.Int("failed", failed))
	}(c.Sender().ID)

	return nil
}

func (h *Handler) executeAdminGiveCoinsAll(c tele.Context, amount int) error {
	ids, err := h.users.GetAllTelegramIDs()
	if err != nil {
		h.log.Error("give_coins_all: failed to load users", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	affected, err := h.users.GiveCoinsToAll(amount)
	if err != nil {
		h.log.Error("give_coins_all failed", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	h.notifyCoinGift(ids, amount)
	return c.Send(fmt.Sprintf(messages.AdminCoinsGivenAll, amount, affected))
}

func (h *Handler) executeAdminGiveCoinsPoor(c tele.Context, amount int) error {
	ids, err := h.users.GetTelegramIDsWithCoinsBelow(poorCoinThreshold)
	if err != nil {
		h.log.Error("give_coins_poor: failed to load users", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	affected, err := h.users.GiveCoinsToPoor(amount, poorCoinThreshold)
	if err != nil {
		h.log.Error("give_coins_poor failed", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	h.notifyCoinGift(ids, amount)
	return c.Send(fmt.Sprintf(messages.AdminCoinsGivenPoor, amount, affected))
}

func (h *Handler) setBanned(c tele.Context, ref string, banned bool, confirm, notify string) error {
	targetID := strings.TrimSpace(strings.TrimPrefix(ref, "/user_"))
	if targetID == "" {
		return c.Send(messages.AdminUserNotFound)
	}
	target, err := h.users.GetByID(targetID)
	if err != nil {
		return c.Send(messages.AdminUserNotFound)
	}

	if err := h.users.SetBanned(target.TelegramID, banned); err != nil {
		h.log.Error("set banned failed", zap.Error(err), zap.String("target", target.ID))
		return c.Send(messages.AdminOperationFailed)
	}

	// Best-effort notification to the affected user.
	h.bot.Send(&tele.User{ID: target.TelegramID}, notify)

	h.log.Info("admin ban state changed",
		zap.Int64("admin", c.Sender().ID),
		zap.String("target", target.ID),
		zap.Bool("banned", banned))

	return c.Send(fmt.Sprintf(confirm, userPublicRef(target.ID)))
}

func (h *Handler) notifyCoinGift(ids []int64, amount int) {
	go func() {
		text := fmt.Sprintf(messages.AdminCoinsGiftedUser, amount)
		for _, id := range ids {
			h.bot.Send(&tele.User{ID: id}, text)
			time.Sleep(broadcastDelay)
		}
	}()
}

func (h *Handler) setAdminPending(adminID int64, pending *AdminPendingAction) {
	h.adminMu.Lock()
	defer h.adminMu.Unlock()
	h.adminPending[adminID] = pending
}

func (h *Handler) getAdminPending(adminID int64) *AdminPendingAction {
	h.adminMu.Lock()
	defer h.adminMu.Unlock()
	pending := h.adminPending[adminID]
	if pending == nil {
		return nil
	}
	cp := *pending
	return &cp
}

func (h *Handler) clearAdminPending(adminID int64) {
	h.adminMu.Lock()
	defer h.adminMu.Unlock()
	delete(h.adminPending, adminID)
}

func isValidAdminCommand(command string) bool {
	switch command {
	case adminCmdBroadcast, adminCmdGiveCoinsAll, adminCmdGiveCoinsPoor, adminCmdBan, adminCmdUnban:
		return true
	default:
		return false
	}
}

func adminInputPrompt(command string) string {
	switch command {
	case adminCmdBroadcast:
		return messages.AdminBroadcastUsage
	case adminCmdGiveCoinsAll:
		return fmt.Sprintf(messages.AdminGiveCoinsUsage, "/give_coins_all")
	case adminCmdGiveCoinsPoor:
		return fmt.Sprintf(messages.AdminGiveCoinsUsage, "/give_coins_poor")
	case adminCmdBan:
		return messages.AdminBanUsage
	case adminCmdUnban:
		return messages.AdminUnbanUsage
	default:
		return messages.ErrInvalidCommand
	}
}

func (h *Handler) validateAdminPayload(command, payload string) string {
	payload = strings.TrimSpace(payload)
	switch command {
	case adminCmdBroadcast:
		if payload == "" || len([]rune(payload)) > 4000 {
			return messages.AdminBroadcastUsage
		}
	case adminCmdGiveCoinsAll, adminCmdGiveCoinsPoor:
		if amount, err := parsePositiveAmount(payload); err != nil || amount <= 0 {
			return messages.AdminInvalidAmount
		}
	case adminCmdBan, adminCmdUnban:
		id := strings.TrimSpace(strings.TrimPrefix(payload, "/user_"))
		if id == "" || strings.Contains(id, " ") {
			return messages.AdminUserNotFound
		}
		if _, err := h.users.GetByID(id); err != nil {
			return messages.AdminUserNotFound
		}
	}
	return ""
}

func (h *Handler) adminActionSummary(pending *AdminPendingAction) string {
	switch pending.Command {
	case adminCmdBroadcast:
		return fmt.Sprintf("📣 ارسال پیام همگانی:\n%s", pending.Payload)
	case adminCmdGiveCoinsAll:
		return fmt.Sprintf("💰 اهدای %s سکه به همه کاربران فعال", pending.Payload)
	case adminCmdGiveCoinsPoor:
		return fmt.Sprintf("🪙 اهدای %s سکه به کاربران با کمتر از %d سکه", pending.Payload, poorCoinThreshold)
	case adminCmdBan:
		return fmt.Sprintf("🚫 مسدود کردن کاربر %s", normalizePublicUserRef(pending.Payload))
	case adminCmdUnban:
		return fmt.Sprintf("✅ رفع مسدودی کاربر %s", normalizePublicUserRef(pending.Payload))
	default:
		return messages.ErrInvalidCommand
	}
}

func normalizePublicUserRef(ref string) string {
	return userPublicRef(strings.TrimSpace(strings.TrimPrefix(ref, "/user_")))
}

func userPublicRef(dbID string) string {
	return fmt.Sprintf("/user_%s", dbID)
}

// parsePositiveAmount parses the payload of a coin command. The (0, err)
// return signals "missing argument" so callers can show usage instead.
func parsePositiveAmount(payload string) (int, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return 0, fmt.Errorf("missing amount")
	}
	amount, err := strconv.Atoi(payload)
	if err != nil || amount <= 0 || amount > 1_000_000 {
		return -1, fmt.Errorf("invalid amount")
	}
	return amount, nil
}
