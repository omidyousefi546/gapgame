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

// AdminHelpHandler lists the available admin commands.
func (h *Handler) AdminHelpHandler(c tele.Context) error {
	return c.Send(messages.AdminHelp)
}

// AdminBroadcastHandler sends a message to every user in the database.
// Usage: /broadcast <text>
func (h *Handler) AdminBroadcastHandler(c tele.Context) error {
	text := strings.TrimSpace(c.Message().Payload)
	if text == "" {
		return c.Send(messages.AdminBroadcastUsage)
	}

	ids, err := h.users.GetAllTelegramIDs()
	if err != nil {
		h.log.Error("broadcast: failed to load users", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}

	c.Send(fmt.Sprintf(messages.AdminBroadcastStarted, len(ids)))

	// Send in the background so the admin handler doesn't block the poller.
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
		h.bot.Send(&tele.User{ID: adminID},
			fmt.Sprintf(messages.AdminBroadcastFinished, ok, failed))
		h.log.Info("broadcast finished", zap.Int("ok", ok), zap.Int("failed", failed))
	}(c.Sender().ID)

	return nil
}

// AdminGiveCoinsAllHandler gives coins to all users.
// Usage: /give_coins_all <amount>
func (h *Handler) AdminGiveCoinsAllHandler(c tele.Context) error {
	amount, err := parsePositiveAmount(c.Message().Payload)
	if err != nil {
		if amount == 0 {
			return c.Send(fmt.Sprintf(messages.AdminGiveCoinsUsage, "/give_coins_all"))
		}
		return c.Send(messages.AdminInvalidAmount)
	}

	affected, err := h.users.GiveCoinsToAll(amount)
	if err != nil {
		h.log.Error("give_coins_all failed", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	return c.Send(fmt.Sprintf(messages.AdminCoinsGivenAll, amount, affected))
}

// AdminGiveCoinsPoorHandler gives coins only to users with < 2 coins.
// Usage: /give_coins_poor <amount>
func (h *Handler) AdminGiveCoinsPoorHandler(c tele.Context) error {
	amount, err := parsePositiveAmount(c.Message().Payload)
	if err != nil {
		if amount == 0 {
			return c.Send(fmt.Sprintf(messages.AdminGiveCoinsUsage, "/give_coins_poor"))
		}
		return c.Send(messages.AdminInvalidAmount)
	}

	affected, err := h.users.GiveCoinsToPoor(amount, poorCoinThreshold)
	if err != nil {
		h.log.Error("give_coins_poor failed", zap.Error(err))
		return c.Send(messages.AdminOperationFailed)
	}
	return c.Send(fmt.Sprintf(messages.AdminCoinsGivenPoor, amount, affected))
}

// AdminBanHandler bans a user. Usage: /ban <telegram id | /user_xxx>
func (h *Handler) AdminBanHandler(c tele.Context) error {
	return h.setBanned(c, true, messages.AdminBanUsage, messages.AdminUserBanned, messages.AdminBanNotifyUser)
}

// AdminUnbanHandler unbans a user. Usage: /unban <telegram id | /user_xxx>
func (h *Handler) AdminUnbanHandler(c tele.Context) error {
	return h.setBanned(c, false, messages.AdminUnbanUsage, messages.AdminUserUnbanned, messages.AdminUnbanNotifyUser)
}

func (h *Handler) setBanned(c tele.Context, banned bool, usage, confirm, notify string) error {
	ref := strings.TrimSpace(c.Message().Payload)
	if ref == "" {
		return c.Send(usage)
	}

	telegramID, err := h.users.ResolveTelegramID(ref)
	if err != nil {
		return c.Send(messages.AdminUserNotFound)
	}

	if err := h.users.SetBanned(telegramID, banned); err != nil {
		h.log.Error("set banned failed", zap.Error(err), zap.Int64("target", telegramID))
		return c.Send(messages.AdminOperationFailed)
	}

	// Best-effort notification to the affected user.
	h.bot.Send(&tele.User{ID: telegramID}, notify)

	h.log.Info("admin ban state changed",
		zap.Int64("admin", c.Sender().ID),
		zap.Int64("target", telegramID),
		zap.Bool("banned", banned))

	return c.Send(fmt.Sprintf(confirm, telegramID))
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
