package user

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u *User) error {
	id, err := generateUserID()
	if err != nil {
		return fmt.Errorf("generate user id: %w", err)
	}
	u.ID = id
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Update(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// generateUserID یک ID تصادفی و یکتا تولید می‌کنه
func generateUserID() (string, error) {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	id := make([]rune, 10)
	for i := range b {
		id[i] = alphabet[int(b[i])%len(alphabet)]
	}

	return string(id), nil
}

///////////////////////////////////////////////////////////////

// SearchFilter فیلترهای جستجوی کاربر
type SearchFilter struct {
	Type          string
	Gender        string
	MinAge        int
	MaxAge        int
	Provinces     []string
	ExcludeUserID int64
	NoChat        bool
	NewUsers      bool

	NearbyLat *float64
	NearbyLng *float64
	RadiusKM  float64
	Limit     int
	Offset    int
}

func (r *Repository) SearchUsers(ctx context.Context, f SearchFilter) ([]User, error) {
	users, _, err := r.ExecuteSearch(ctx, f)
	return users, err
}

// SearchUsersWithTotal returns search results with total count
func (r *Repository) SearchUsersWithTotal(ctx context.Context, f SearchFilter) ([]User, int64, error) {
	return r.ExecuteSearch(ctx, f)
}

func calcDistance(lat1, lon1, lat2, lon2 float64) float64 {

	const R = 6371

	dLat := (lat2 - lat1) * math.Pi / 180

	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +

		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

}

// Other User profile Action
// // like
func (r *Repository) ToggleLike(ctx context.Context, likerTelegramID, targetTelegramID int64) (liked bool, err error) {
	// همه عملیات در یک تراکنش
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Like
		err := tx.Where("liker_id = ? AND target_id = ?", likerTelegramID, targetTelegramID).
			First(&existing).Error

		if err == nil {
			// لایک وجود داره → برداشتن
			if err := tx.Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&User{}).
				Where("telegram_id = ?", targetTelegramID).
				UpdateColumn("likes", gorm.Expr("likes - 1")).Error; err != nil {
				return err
			}
			liked = false
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// لایک نداره → اضافه کردن
		if err := tx.Create(&Like{
			LikerID:  likerTelegramID,
			TargetID: targetTelegramID,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).
			Where("telegram_id = ?", targetTelegramID).
			UpdateColumn("likes", gorm.Expr("likes + 1")).Error; err != nil {
			return err
		}
		liked = true
		return nil
	})

	return liked, err
}

//// Contact

func (r *Repository) AddContact(ctx context.Context, ownerID, contactID int64, label string) error {
	return r.db.WithContext(ctx).Create(&Contact{
		OwnerID:   ownerID,
		ContactID: contactID,
		Label:     label,
	}).Error
}

func (r *Repository) RemoveContact(ctx context.Context, ownerID, contactID int64) error {
	return r.db.WithContext(ctx).
		Where("owner_id = ? AND contact_id = ?", ownerID, contactID).
		Delete(&Contact{}).Error
}

func (r *Repository) IsContact(ctx context.Context, ownerID, contactID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Contact{}).
		Where("owner_id = ? AND contact_id = ?", ownerID, contactID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetContacts(ctx context.Context, ownerID int64, offset, limit int) ([]Contact, int64, error) {
	var contacts []Contact
	var total int64

	r.db.WithContext(ctx).Model(&Contact{}).Where("owner_id = ?", ownerID).Count(&total)
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Offset(offset).Limit(limit).
		Find(&contacts).Error
	return contacts, total, err
}

func (r *Repository) DeleteAllContacts(ctx context.Context, ownerID int64) error {
	return r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Delete(&Contact{}).Error
}

//// Block

func (r *Repository) BlockUser(ctx context.Context, blockerID, blockedID int64) error {
	block := Block{BlockerID: blockerID, BlockedID: blockedID}
	return r.db.WithContext(ctx).
		Where(block).
		FirstOrCreate(&block).Error
}

// UnblockUser آنبلاک کردن
func (r *Repository) UnblockUser(ctx context.Context, blockerID, blockedID int64) error {
	return r.db.WithContext(ctx).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&Block{}).Error
}

func (r *Repository) IsBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) GetBlocked(ctx context.Context, blockerID int64, offset, limit int) ([]Block, int64, error) {
	var blocks []Block
	var total int64

	r.db.WithContext(ctx).Model(&Block{}).Where("blocker_id = ?", blockerID).Count(&total)
	err := r.db.WithContext(ctx).
		Where("blocker_id = ?", blockerID).
		Offset(offset).Limit(limit).
		Find(&blocks).Error
	return blocks, total, err
}

func (r *Repository) DeleteAllBlocks(ctx context.Context, blockerID int64) error {
	return r.db.WithContext(ctx).Where("blocker_id = ?", blockerID).Delete(&Block{}).Error
}

//// Report

func (r *Repository) ReportUser(ctx context.Context, reporterID, targetID int64, reason string) error {
	return r.db.WithContext(ctx).Create(&Report{
		ReporterID: reporterID,
		TargetID:   targetID,
		Reason:     reason,
	}).Error
}

// My profile Action

// GetLikedBy کسایی که این کاربر رو لایک کردن با pagination
func (r *Repository) GetLikedBy(ctx context.Context, telegramID int64, offset, limit int) ([]User, int64, error) {
	var users []User
	var total int64

	r.db.WithContext(ctx).
		Model(&Like{}).
		Where("target_id = ?", telegramID).
		Count(&total)

	err := r.db.WithContext(ctx).
		Select("users.*").
		Joins("JOIN likes ON likes.liker_id = users.telegram_id").
		Where("likes.target_id = ?", telegramID).
		Offset(offset).
		Limit(limit).
		Find(&users).Error

	return users, total, err
}

// GetLikesByTelegramID لیست کسایی که این کاربر لایک کرده
func (r *Repository) GetLikesByTelegramID(ctx context.Context, telegramID int64) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Select("users.*").
		Joins("JOIN likes ON likes.target_id = users.telegram_id").
		Where("likes.liker_id = ?", telegramID).
		Find(&users).Error
	return users, err
}

// GetContactsByTelegramID لیست مخاطبین
// func (r *Repository) GetContactsByTelegramID(ctx context.Context, telegramID int64) ([]User, error) {
// 	var users []User
// 	err := r.db.WithContext(ctx).
// 		Select("users.*").
// 		Joins("JOIN contacts ON contacts.contact_id = users.telegram_id").
// 		Where("contacts.owner_id = ?", telegramID).
// 		Find(&users).Error
// 	return users, err
// }

// GetBlockedByTelegramID لیست بلاک شده‌ها
func (r *Repository) GetBlockedByTelegramID(ctx context.Context, telegramID int64) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Select("users.*").
		Joins("JOIN blocks ON blocks.blocked_id = users.telegram_id").
		Where("blocks.blocker_id = ?", telegramID).
		Find(&users).Error
	return users, err
}

// ToggleSilent سایلنت رو toggle میکنه — باید فیلد Silent به User اضافه بشه
// func (r *Repository) ToggleSilent(ctx context.Context, telegramID int64) (bool, error) {
// 	var u User
// 	err := r.db.WithContext(ctx).
// 		Where("telegram_id = ?", telegramID).
// 		First(&u).Error
// 	if err != nil {
// 		return false, err
// 	}
// 	u.Silent = !u.Silent
// 	err = r.db.WithContext(ctx).
// 		Model(&u).
// 		Update("silent", u.Silent).Error
// 	return u.Silent, err
// }

// last seen
func (r *Repository) GetAllUserIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).
		Model(&User{}).
		Pluck("telegram_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("GetAllUserIDs: %w", err)
	}
	return ids, nil
}
func (r *Repository) BulkUpdateLastSeen(ctx context.Context, data map[int64]time.Time) error {
	if len(data) == 0 {
		return nil
	}

	// تشخیص دیتابیس
	dialect := r.db.Dialector.Name()

	if dialect == "postgres" {
		// PostgreSQL: unnest
		type row struct {
			ID int64
			T  time.Time
		}
		rows := make([]row, 0, len(data))
		for id, t := range data {
			rows = append(rows, row{id, t})
		}

		ids := make([]int64, len(rows))
		times := make([]time.Time, len(rows))
		for i, r := range rows {
			ids[i] = r.ID
			times[i] = r.T
		}

		return r.db.WithContext(ctx).Exec(`
			UPDATE users SET last_seen_at = v.t
			FROM (SELECT unnest(?::bigint[]) AS id, unnest(?::timestamptz[]) AS t) AS v
			WHERE users.telegram_id = v.id
		`, pq.Array(ids), pq.Array(times)).Error
	}

	// SQLite: CASE WHEN
	if len(data) > 500 {
		// chunk کن برای SQLite
		chunks := make([]map[int64]time.Time, 0)
		chunk := make(map[int64]time.Time)
		i := 0
		for id, t := range data {
			chunk[id] = t
			i++
			if i%500 == 0 {
				chunks = append(chunks, chunk)
				chunk = make(map[int64]time.Time)
			}
		}
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		for _, c := range chunks {
			if err := r.bulkUpdateSQLite(ctx, c); err != nil {
				return err
			}
		}
		return nil
	}

	return r.bulkUpdateSQLite(ctx, data)
}

func (r *Repository) bulkUpdateSQLite(ctx context.Context, data map[int64]time.Time) error {
	ids := make([]int64, 0, len(data))
	for id := range data {
		ids = append(ids, id)
	}

	query := "UPDATE users SET last_seen_at = CASE telegram_id "
	args := make([]interface{}, 0, len(data)*2)

	for _, id := range ids {
		query += "WHEN ? THEN ? "
		args = append(args, id, data[id])
	}

	query += "END WHERE telegram_id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"
	args = append(args, ids[0])
	for i := 1; i < len(ids); i++ {
		args = append(args, ids[i])
	}

	return r.db.WithContext(ctx).Exec(query, args...).Error
}

// ─── Admin operations ─────────────────────────────────────────

// GetAllTelegramIDs returns the Telegram IDs of every (non-banned) user.
// Used by the admin broadcast command.
func (r *Repository) GetAllTelegramIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).
		Model(&User{}).
		Where("banned = ?", false).
		Pluck("telegram_id", &ids).Error
	return ids, err
}

// AddCoinsToAll atomically adds `amount` coins to every user and returns the
// number of affected rows.
func (r *Repository) AddCoinsToAll(ctx context.Context, amount int) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&User{}).
		Where("1 = 1").
		UpdateColumn("coins", gorm.Expr("coins + ?", amount))
	return res.RowsAffected, res.Error
}

// AddCoinsToPoor adds `amount` coins only to users whose balance is below
// `threshold` (e.g. < 2 coins) and returns the number of affected rows.
func (r *Repository) AddCoinsToPoor(ctx context.Context, amount, threshold int) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&User{}).
		Where("coins < ?", threshold).
		UpdateColumn("coins", gorm.Expr("coins + ?", amount))
	return res.RowsAffected, res.Error
}

// SetBanned bans or unbans a user by Telegram ID.
func (r *Repository) SetBanned(ctx context.Context, telegramID int64, banned bool) error {
	res := r.db.WithContext(ctx).
		Model(&User{}).
		Where("telegram_id = ?", telegramID).
		UpdateColumn("banned", banned)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// IsBanned reports whether the user with the given Telegram ID is banned.
func (r *Repository) IsBanned(ctx context.Context, telegramID int64) (bool, error) {
	var banned bool
	err := r.db.WithContext(ctx).
		Model(&User{}).
		Where("telegram_id = ?", telegramID).
		Pluck("banned", &banned).Error
	return banned, err
}
