package user

import "time"

type UserState string

const (
	StateProfileBasic    UserState = "profile_basic"
	StateProfileOptional UserState = "profile_optional"
	StateMenu            UserState = "menu"
	StateChat            UserState = "chat"
)

type Gender string

const (
	Male   Gender = "male"
	Female Gender = "female"
)

type ProfileState string

const (
	StateNeedGender   ProfileState = "need_gender"
	StateNeedAge      ProfileState = "need_age"
	StateNeedProvince ProfileState = "need_province"
	StateComplete     ProfileState = "complete"
)

type User struct {
	ID                   string `gorm:"primaryKey"`
	TelegramID           int64  `gorm:"uniqueIndex;not null"`
	Name                 string `gorm:"type:varchar(100)"`
	Gender               Gender `gorm:"type:varchar(10)"`
	Age                  *int
	Province             string `gorm:"type:varchar(100)"`
	City                 string `gorm:"type:varchar(100)"`
	ProfilePhoto         string `gorm:"type:text"`
	Coins                int    `gorm:"default:0"`
	Likes                int    `gorm:"default:0"`
	Latitude             *float64
	Longitude            *float64
	ProfileState         ProfileState `gorm:"type:varchar(20);default:'need_gender'"`
	ReceivedProfileBonus bool  `gorm:"default:false"`
	InviteCount          int
	DisableLikes         bool `gorm:"default:false"`

	SilentUntil *time.Time `gorm:"default:null"`
	LastSeenAt  time.Time  `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (u *User) IsProfileComplete() bool {
	return u.Gender != "" && u.Age != nil && u.Province != ""
}

// GetMissingOptionalFields فیلدهای اختیاری که تکمیل نشدن رو برمی‌گردونه
func (u *User) GetMissingOptionalFields() []string {
	var missing []string
	if u.Name == "" {
		missing = append(missing, "نام")
	}
	if u.City == "" {
		missing = append(missing, "شهر")
	}
	if u.ProfilePhoto == "" {
		missing = append(missing, "عکس پروفایل")
	}
	if u.Latitude == nil || u.Longitude == nil {
		missing = append(missing, "موقعیت مکانی")
	}
	return missing
}

func (u *User) SafeAge() int {
	if u.Age == nil {
		return 0
	}
	return *u.Age
}

func (u *User) GetPhoto() string {
	if u.ProfilePhoto == "" {
		if u.Gender == Male {
			return "1195336774:869497621718769409:1:d71a0507c98b48f3fac5bd920215e8091135658b4958d6b9"
		}
		return "1195336774:6925235746642861827:1:d71a0507c98b48f3fac5bd920215e809d4f1ab33f2e83811"
	}
	return u.ProfilePhoto
}

type Like struct {
	ID       uint  `gorm:"primaryKey"`
	LikerID  int64 `gorm:"uniqueIndex:idx_like"`
	TargetID int64 `gorm:"uniqueIndex:idx_like"`
}
type Contact struct {
	ID        uint   `gorm:"primaryKey"`
	OwnerID   int64  `gorm:"uniqueIndex:idx_contact"`
	ContactID int64  `gorm:"uniqueIndex:idx_contact"`
	Label     string `gorm:"not null"` // اسمی که owner انتخاب میکنه
}
type Block struct {
	ID        uint  `gorm:"primaryKey"`
	BlockerID int64 `gorm:"uniqueIndex:idx_block"`
	BlockedID int64 `gorm:"uniqueIndex:idx_block"`
}

type Report struct {
	ID         uint  `gorm:"primaryKey"`
	ReporterID int64 `gorm:"index"`
	TargetID   int64 `gorm:"index"`
	Reason     string
	CreatedAt  time.Time
}
