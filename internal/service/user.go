package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/internal/utils"
	"GapGame/pkg/messages"
)

var (
	ErrSelfReferral = errors.New("cannot refer yourself")

	ErrInvalidAge = errors.New("invalid age")

	ErrInvalidProvince = errors.New("invalid province")

	ErrInvalidGender = errors.New("invalid gender")

	ErrInvalidCity = errors.New("invalid city")

	ErrInvalidName = errors.New("invalid name")
)

type UserService struct {
	repo    *user.Repository
	repoOpt *user.RepositoryOptimizations
	session *session.Manager
}

func NewUserService(repo *user.Repository, repoOpt *user.RepositoryOptimizations, sm *session.Manager) *UserService {
	return &UserService{repo: repo, repoOpt: repoOpt, session: sm}
}

// GetOrCreate finds or creates user by telegram ID

func (s *UserService) GetOrCreate(telegramID int64) (*user.User, bool, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	u, err := s.repo.GetByTelegramID(ctx, telegramID)

	if err == nil {

		return u, false, nil //user, isNew, error
	}

	u = &user.User{

		TelegramID:   telegramID,
		ProfileState: user.StateNeedGender,
		Coins:        0,
	}

	ctx2, cancel2 := utils.NewRequestContext()
	defer cancel2()
	if err := s.repo.Create(ctx2, u); err != nil {

		return nil, false, err
	}

	return u, true, nil

}

func (s *UserService) GetByTelegramID(telegramID int64) (*user.User, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetByTelegramID(ctx, telegramID)
}

// Get user by ID

func (s *UserService) GetByID(id string) (*user.User, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetByID(ctx, id)
}

// update
func (s *UserService) Save(u *user.User) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.Update(ctx, u)
}

// HandleReferral processes referral link and awards coins

func (s *UserService) HandleReferral(newUser *user.User, referrerID string) (*user.User, error) {

	if referrerID == "" || referrerID == newUser.ID {

		return nil, ErrSelfReferral
	}

	referrer, err := s.GetByID(referrerID)

	if err != nil {

		return nil, fmt.Errorf("referrer not found: %w", err)
	}

	// Award coins to referrer
	value, _ := strconv.ParseInt(messages.EnterUserCoin, 10, 64)
	referrer.Coins += int(value)

	referrer.InviteCount++

	ctx2, cancel2 := utils.NewRequestContext()
	defer cancel2()
	if err := s.repo.Update(ctx2, referrer); err != nil {

		return nil, err
	}

	return referrer, nil

}

// CompleteProfileField updates a profile field during completion flow

func (s *UserService) CompleteProfileField(u *user.User, field, value string) error {

	switch field {

	case "gender":

		if value != string(user.Male) && value != string(user.Female) {
			return errors.New("invalid gender")
		}
		u.Gender = user.Gender(value)
		u.ProfileState = user.StateNeedAge
	case "age":
		value = NormalizeDigits(value)
		age, err := strconv.Atoi(value)
		if err != nil || age < 13 || age > 100 {
			return ErrInvalidAge
		}
		u.Age = &age
		u.ProfileState = user.StateNeedProvince
	case "province":

		if !utils.ContainsProvince(value) {
			return ErrInvalidProvince
		}

		u.Province = value
		u.ProfileState = user.StateComplete
	default:

		return fmt.Errorf("unknown field: %s", field)
	}
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	return s.repo.Update(ctx, u)

}

func NormalizeDigits(s string) string {
	r := strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4", "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4", "٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	)
	return r.Replace(s)
}

// StartOptionalProfile awards bonus and sets state

func (s *UserService) StartOptionalProfile(u *user.User) error {

	if u.ReceivedProfileBonus {

		return nil
	}

	u.Coins += 5

	u.ReceivedProfileBonus = true

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.Update(ctx, u)

}

// UpdateOptionalField updates name, city, photo, or GPS

func (s *UserService) UpdateOptionalField(u *user.User, field, value string) error {

	switch field {

	case "gender":

		if value != string(user.Male) && value != string(user.Female) {
			return ErrInvalidGender
		}
		u.Gender = user.Gender(value)
		u.ProfileState = user.StateNeedAge

	case "age":
		value = NormalizeDigits(value)
		age, err := strconv.Atoi(value)
		if err != nil || age < 13 || age > 100 {
			return ErrInvalidAge
		}
		u.Age = &age
		u.ProfileState = user.StateNeedProvince

	case "province":

		if !utils.ContainsProvince(value) {
			return ErrInvalidProvince
		}
		u.Province = value
		u.ProfileState = user.StateComplete
	case "name":

		if !isValidPersian(value) || len([]rune(value)) > 20 {
			return ErrInvalidName
		}
		u.Name = value
	case "city":

		if !isValidPersian(value) {
			return ErrInvalidCity
		}
		u.City = value
	case "photo":

		u.ProfilePhoto = value
	// case "gps":

	// 	parts := strings.Split(value, ",")
	// 	if len(parts) != 2 {
	// 		return errors.New("invalid GPS format")
	// 	}
	// 	lat, err1 := strconv.ParseFloat(parts[0], 64)
	// 	lon, err2 := strconv.ParseFloat(parts[1], 64)
	// 	if err1 != nil || err2 != nil {
	// 		return errors.New("invalid GPS coordinates")
	// 	}
	// 	u.Latitude = &lat
	// 	u.Longitude = &lon
	default:

		return fmt.Errorf("unknown field: %s", field)
	}

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.Update(ctx, u)

}

func (s *UserService) UpdateGPS(u *user.User, lat, lon float64) error {
	u.Latitude = &lat
	u.Longitude = &lon

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.Update(ctx, u)
}

// AwardCoins adds coins to user

func (s *UserService) AwardCoins(userID string, amount int, reason string) error {
	u, err := s.GetByID(userID)

	if err != nil {

		return err
	}

	u.Coins += amount

	ctx2, cancel2 := utils.NewRequestContext()
	defer cancel2()
	return s.repo.Update(ctx2, u)

}

// ToggleLike handles like/unlike

func (s *UserService) ToggleLike(fromID, toID int64) (bool, error) {

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.ToggleLike(ctx, fromID, toID)

}

// AddContact adds user to contacts

// func (s *UserService) AddContact(fromID, toID int64) error {
// 	ctx, cancel := utils.NewRequestContext()
// 	defer cancel()
// 	return s.repo.AddContact(ctx, fromID, toID)

// }

// BlockUser blocks a user

func (s *UserService) BlockUser(blockerID, blockedID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	return s.repo.BlockUser(ctx, blockerID, blockedID)

}

func (s *UserService) UnblockUser(blockerID, blockedID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.UnblockUser(ctx, blockerID, blockedID)
}

func (s *UserService) IsBlocked(blockerID, blockedID int64) (bool, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.IsBlocked(ctx, blockerID, blockedID)
}

func (s *UserService) GetBlocked(blockerID int64, offset, limit int) ([]user.Block, int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetBlocked(ctx, blockerID, offset, limit)
}

func (s *UserService) DeleteAllBlocks(blockerID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.DeleteAllBlocks(ctx, blockerID)
}

// ReportUser reports a user

func (s *UserService) ReportUser(fromID, toID int64, reason string) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.ReportUser(ctx, fromID, toID, reason)

}

//// Mu profile Actions

func (s *UserService) GetLikedBy(telegramID int64, page, pageSize int) ([]user.User, int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	offset := (page - 1) * pageSize
	return s.repo.GetLikedBy(ctx, telegramID, offset, pageSize)
}

func (s *UserService) GetLikes(telegramID int64) ([]user.User, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetLikesByTelegramID(ctx, telegramID)
}

func (s *UserService) AddContact(ownerID, contactID int64, label string) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.AddContact(ctx, ownerID, contactID, label)
}

func (s *UserService) RemoveContact(ownerID, contactID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.RemoveContact(ctx, ownerID, contactID)
}

func (s *UserService) IsContact(ownerID, contactID int64) (bool, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.IsContact(ctx, ownerID, contactID)
}

func (s *UserService) GetContacts(ownerID int64, offset, limit int) ([]user.Contact, int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetContacts(ctx, ownerID, offset, limit)
}

// BatchGetByTelegramIDs returns multiple users in a single query.
// Used to avoid N+1 lookups when rendering paginated lists.
func (s *UserService) BatchGetByTelegramIDs(ids []int64) ([]user.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repoOpt.BatchGetByTelegramIDs(ctx, ids)
}
func (s *UserService) DeleteAllContacts(ownerID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.DeleteAllContacts(ctx, ownerID)
}

// func (s *UserService) ToggleSilent(telegramID int64) (bool, error) {
// 	ctx, cancel := utils.NewRequestContext()
// 	defer cancel()
// 	return s.repo.ToggleSilent(ctx, telegramID)
// }

// SearchUsers searches for users with filters

func (s *UserService) SearchUsers(filter *user.SearchFilter) ([]user.User, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	if filter.Limit <= 0 {
		filter.Limit = 10 // default
	}

	return s.repo.SearchUsers(ctx, *filter)

}

// GetUserByID retrieves user by ID

// func (s *UserService) GetUserByID(id string) (*user.User, error) {
// 	ctx, cancel := NewRequestContext()
// 	defer cancel()
// 	return s.repo.GetByID(ctx, id)

// }

// UpdateLastSeen updates user’s last seen timestamp

func (s *UserService) UpdateLastSeen(telegram_id int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.session.UpdateLastSeen(ctx, telegram_id)

}

// GetLastSeen retrieves last seen time

func (s *UserService) GetLastSeen(telegram_id int64) (time.Time, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.session.GetLastSeen(ctx, telegram_id)

}

// IsOnline checks if user is online

// func (s *UserService) IsOnline(telegram_id int64) bool {
// 	ctx, cancel := NewRequestContext()
// 	defer cancel()
// 	return s.session.IsOnline(ctx, telegram_id)

// }

// isValidPersian checks if string contains only Persian characters and spaces

func isValidPersian(s string) bool {

	if len(strings.TrimSpace(s)) == 0 {

		return false
	}

	for _, r := range s {

		if !((r >= 0x0600 && r <= 0x06FF) || r == ' ') {
			return false
		}
	}

	return true

}

// SetDMTarget

func (s *UserService) SetDMTarget(userID, targetID int64) error {

	return s.session.SetDMTarget(userID, targetID)
}

// GetDMTarget

func (s *UserService) GetDMTarget(userID int64) (int64, error) {

	return s.session.GetDMTarget(userID)
}

// DeleteDMTarget

func (s *UserService) DeleteDMTarget(userID int64) error {

	return s.session.ClearDMTarget(userID)
}

// ─── Notify Online ────────────────────────────────────────────

func (s *UserService) AddOnlineNotify(targetID, notifyTelegramID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.session.AddOnlineNotify(ctx, targetID, notifyTelegramID)
}

func (s *UserService) PopOnlineNotifyList(targetID int64) ([]string, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.session.PopOnlineNotifyList(ctx, targetID)
}

// chat

func (s *UserService) AwardCoinsByTelegramID(telegramID int64, amount int, reason string) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	u, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	u.Coins += amount
	ctx2, cancel2 := utils.NewRequestContext()
	defer cancel2()
	return s.repo.Update(ctx2, u)
}

func (s *UserService) DeductCoins(telegramID int64, amount int) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	u, err := s.repo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return err
	}
	if u.Coins < amount {
		return fmt.Errorf("insufficient coins: have %d, need %d", u.Coins, amount)
	}
	u.Coins -= amount
	ctx2, cancel2 := utils.NewRequestContext()
	defer cancel2()
	return s.repo.Update(ctx2, u)
}

// ─── Admin operations ────────────────────────────────────────

// GetAllTelegramIDs returns every active user's Telegram ID (for broadcast).
func (s *UserService) GetAllTelegramIDs() ([]int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.GetAllTelegramIDs(ctx)
}

// GiveCoinsToAll adds coins to every user. Returns affected user count.
func (s *UserService) GiveCoinsToAll(amount int) (int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.AddCoinsToAll(ctx, amount)
}

// GiveCoinsToPoor adds coins only to users with fewer than `threshold` coins.
func (s *UserService) GiveCoinsToPoor(amount, threshold int) (int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.AddCoinsToPoor(ctx, amount, threshold)
}

// SetBanned bans or unbans a user by Telegram ID.
func (s *UserService) SetBanned(telegramID int64, banned bool) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.SetBanned(ctx, telegramID, banned)
}

// IsBanned reports whether a user is banned.
func (s *UserService) IsBanned(telegramID int64) (bool, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return s.repo.IsBanned(ctx, telegramID)
}

// ResolveTelegramID accepts either a numeric Telegram ID or a short bot ID
// (the /user_xxx form) and resolves it to a Telegram ID.
func (s *UserService) ResolveTelegramID(ref string) (int64, error) {
	ref = strings.TrimSpace(strings.TrimPrefix(ref, "/user_"))
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		// Numeric — could be a Telegram ID already.
		if _, gerr := s.GetByTelegramID(id); gerr == nil {
			return id, nil
		}
	}
	u, err := s.GetByID(ref)
	if err != nil {
		return 0, err
	}
	return u.TelegramID, nil
}
