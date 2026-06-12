package bot

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"time"

	"GapGame/internal/service"
	"GapGame/internal/utils"
	"GapGame/pkg/messages"

	"GapGame/internal/session"

	"GapGame/internal/user"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (h *Handler) ProfileHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrUserFetch)
	}

	if c.Callback() != nil {

		c.Respond()
	}
	return h.sendProfileMessage(c, u, u, true)

}

func (h *Handler) ShowUserProfile(c tele.Context, viewer *user.User, target *user.User) error {

	return h.sendProfileMessage(c, viewer, target, false)

}

func (h *Handler) sendProfileMessage(c tele.Context, viewer *user.User, target *user.User, isSelf bool) error {

	photo := target.GetPhoto()

	var kb *tele.ReplyMarkup
	var text string
	var isOnline bool
	if isSelf {
		_, text = h.buildProfileText(viewer, target)
		kb = ProfileInlineKeyboard(viewer)
	} else {

		isOnline, text = h.buildProfileText(viewer, target)

		kb = ProfileActionKeyboard(h.users, viewer, target, isOnline)
	}

	return editOrSend(c, &tele.Photo{File: tele.File{FileID: photo}, Caption: text}, kb, tele.ModeHTML)

}

func (h *Handler) buildProfileText(viewer *user.User, u *user.User) (bool, string) { //isflag, text

	name := "❓"

	if u.Name != "" {

		name = u.Name
	}

	gender := "❓"

	switch u.Gender {
	case user.Male:
		gender = "👨 مرد"
	case user.Female:
		gender = "👩 زن"
	}

	age := u.SafeAge()

	province := "❓"

	if u.Province != "" {

		province = u.Province
	}

	city := "❓"

	if u.City != "" {

		city = u.City
	}

	lastSeen := h.formatLastSeenRealTime(u.TelegramID)

	distance := "🏁 فاصله از شما: "

	if viewer.TelegramID == u.TelegramID {
		return true, fmt.Sprintf(
			"• نام: %s\n• جنسیت: %s\n• استان: %s\n• شهر: %s\n• سن: %d\n\n❤️ لایک ها : %d\n\n⏳ %s\n\n🆔 آیدی : /user_%s",
			name, gender, province, city, age, u.Likes, lastSeen, u.ID,
		)
	} else {
		if viewer.Latitude == nil {
			distance = distance + "موقعیت شما ثبت نشده است"
		} else if u.Latitude == nil {
			distance = distance + "موقعیت کاربر ثبت نشده است"
		} else {
			d := calcDistance(*viewer.Latitude, *viewer.Longitude, *u.Latitude, *u.Longitude)
			distance = distance + fmt.Sprintf("%.0f کیلومتر", d)
		}

		isOnline := lastSeen == "👀 هم اکنون آنلاین"
		return isOnline, fmt.Sprintf(
			"• نام: %s\n• جنسیت: %s\n• استان: %s\n• شهر: %s\n• سن: %d\n\n❤️ لایک ها : %d\n\n⏳ %s\n\n🆔 آیدی : /user_%s \n%s",
			name, gender, province, city, age, u.Likes, lastSeen, u.ID, distance,
		)
	}

}

func (h *Handler) formatLastSeenRealTime(telegram_id int64) string {

	t, err := h.users.GetLastSeen(telegram_id)

	if err != nil {

		return "بازدید خیلی وقت پیش"
	}

	diff := time.Since(t)

	switch {

	case diff < 2*time.Minute:
		return "👀 هم اکنون آنلاین"

	case diff < time.Hour:

		return fmt.Sprintf("⏳ %d دقیقه پیش", int(diff.Minutes()))
	case diff < 24*time.Hour:

		return fmt.Sprintf("⏳ %d ساعت پیش", int(diff.Hours()))

	case diff < 24*5*time.Hour:
		return fmt.Sprintf("⏳ %d روز پیش", int(diff.Hours()/24))

	default:
		return "بازدید خیلی وقت پیش"
	}

}

func formatLastSeen(t time.Time) string {

	// t, err := h.users.GetLastSeen(telegram_id)

	// if err != nil {

	// 	return "بازدید خیلی وقت پیش"
	// }

	diff := time.Since(t)

	switch {

	case diff < 2*time.Minute:
		return "👀 هم اکنون آنلاین"

	case diff < time.Hour:

		return fmt.Sprintf("⏳ %d دقیقه پیش", int(diff.Minutes()))
	case diff < 24*time.Hour:

		return fmt.Sprintf("⏳ %d ساعت پیش", int(diff.Hours()))

	case diff < 24*5*time.Hour:
		return fmt.Sprintf("⏳ %d روز پیش", int(diff.Hours()/24))

	default:
		return "بازدید خیلی وقت پیش"
	}

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

// — Profile Actions —
func (h *Handler) LikeHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	targetID, _ := strconv.ParseInt(c.Callback().Data, 10, 64)

	target, _, err := h.users.GetOrCreate(targetID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	// چک DisableLikes - نه Silent
	if target.DisableLikes {
		return c.Respond(&tele.CallbackResponse{Text: "❌ این کاربر بخش لایک را غیرفعال کرده"})
	}

	liked, err := h.users.ToggleLike(u.TelegramID, targetID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ثبت لایک"})
	}

	if liked {
		if _, err := h.bot.Send(&tele.User{ID: targetID},
			fmt.Sprintf("❤️ شما را لایک کرد /user_%s", u.ID)); err != nil {
			h.log.Error("failed to send like notification", zap.Error(err))
		}
		return c.Respond(&tele.CallbackResponse{Text: "❤️ لایک شد!"})
	}

	return c.Respond(&tele.CallbackResponse{Text: "💔 لایک برداشته شد"})
}

func (h *Handler) DMHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	targetIDstring := c.Data()
	targetID, _ := strconv.ParseInt(targetIDstring, 10, 64)

	if err != nil {

		return editOrSend(c, messages.ErrUserNotFound, MainMenuKeyboard())
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateDM); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	h.users.SetDMTarget(u.TelegramID, targetID)

	c.Respond()

	return editOrSend(c, "✉️ پیام خود را بنویسید:", CancelKeyboard())

}

func (h *Handler) ChatRequestHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	targetID, err := strconv.ParseInt(c.Data(), 10, 64)
	if err != nil {
		return editOrSend(c, messages.ErrUserNotFound, MainMenuKeyboard())
	}

	target, _, err := h.users.GetOrCreate(targetID)
	if err != nil {
		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if IsSilent(target) {
		return c.Respond(&tele.CallbackResponse{Text: "🔕 این کاربر در حالت سایلنت است"})
	}

	c.Respond()
	senderID := fmt.Sprintf("/user_%s", u.ID)
	if _, err := h.bot.Send(&tele.User{ID: targetID},
		fmt.Sprintf("💬 کاربر %s درخواست چت دارد!\n\nID: %s", u.Name, senderID),
		ChatRequestKeyboard(u)); err != nil {
		h.log.Error("failed to send chat request", zap.Error(err))
	}
	return editOrSend(c, "✅ درخواست چت ارسال شد.")
}

func (h *Handler) ReportHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	targetIDstring := c.Data()
	targetID, err := strconv.ParseInt(targetIDstring, 10, 64)

	if err != nil {
		return editOrSend(c, messages.ErrUserNotFound, MainMenuKeyboard())
	}

	if err := h.users.ReportUser(u.TelegramID, targetID, ""); err != nil {

		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در گزارش"})
	}

	return c.Respond(&tele.CallbackResponse{Text: "⚠️ گزارش ثبت شد. ممنون"})

}

func (h *Handler) NotifyOnlineHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	targetIDstring := c.Data()
	targetID, err := strconv.ParseInt(targetIDstring, 10, 64)

	if err := h.users.AddOnlineNotify(targetID, u.TelegramID); err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	return c.Respond(&tele.CallbackResponse{Text: "🔔 وقتی آنلاین شد خبرت می‌دم!"})

}

// — Edit Profile —

func (h *Handler) EditProfileHandler(c tele.Context) error {

	msg := c.Message()
	c.Respond()

	kb := EditProfileKeyboard()
	msg, err := h.bot.Edit(msg, kb)
	return err
}

func (h *Handler) BackToEditProfileHandler(c tele.Context) error {
	return c.Reply(messages.BackToMenu, MainMenuKeyboard(), tele.ModeHTML)
}

func (h *Handler) EditNameHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if c.Callback() != nil {
		c.Respond()
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateEditName); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskName, MainMenuKeyboard())

}

func (h *Handler) EditGenderHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}
	if c.Callback() != nil {
		c.Respond()
	}
	if err := h.redis.SetUserState(u.TelegramID, session.StateEditGender); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskGender, GenderEditKeyboard())

}

func (h *Handler) SetGenderHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	gender := string(user.Female)

	if c.Callback().Unique == "set_gender_male" {

		gender = string(user.Male)
	}

	if err := h.users.UpdateOptionalField(u, "gender", gender); err != nil {

		return c.Respond(&tele.CallbackResponse{Text: messages.ErrSetGender})
	}

	if c.Callback() != nil {
		c.Respond()
	}

	return editOrSend(c, messages.UpdateSuccessful, MainMenuKeyboard())

}

func (h *Handler) EditAgeHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if c.Callback() != nil {
		c.Respond()
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateEditAge); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskAge, MainMenuKeyboard())

}

func (h *Handler) EditCityHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if c.Callback() != nil {
		c.Respond()
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateEditCity); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskCity, MainMenuKeyboard())

}

func (h *Handler) EditProvinceHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if c.Callback() != nil {
		c.Respond()
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateEditProvince); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskProvince, ProvinceKeyboard())

}

func (h *Handler) EditPhotoHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	if c.Callback() != nil {
		c.Respond()
	}

	if err := h.redis.SetUserState(u.TelegramID, session.StateEditPhoto); err != nil {
		h.log.Error("failed to set user state", zap.Error(err))
		return editOrSend(c, messages.ErrSystem, MainMenuKeyboard())
	}

	return editOrSend(c, messages.EditAskPhoto, MainMenuKeyboard())

}

func (h *Handler) EditGPSHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}
	h.redis.SetUserState(u.TelegramID, session.StateEditGPS)
	editOrSend(c, messages.GpsMsg1)

	return editOrSend(c, messages.GpsMsg2, LocationKeyboard())

}

func (h *Handler) HandleEditText(c tele.Context, u *user.User, field, value string) error {

	if err := h.users.UpdateOptionalField(u, field, value); err != nil {

		switch err {
		case service.ErrInvalidName:
			return editOrSend(c, messages.ErrNameInvalid, CancelKeyboard())
		case service.ErrInvalidAge:
			return editOrSend(c, messages.ErrInvalidAgeMsg, CancelKeyboard())
		case service.ErrInvalidCity:
			return editOrSend(c, messages.ErrCityInvalid, CancelKeyboard())
		case service.ErrInvalidProvince:
			return editOrSend(c, "❌ نام استان باید از منو زیر انتخاب شود", CancelKeyboard())
		case service.ErrInvalidGender:
			return editOrSend(c, "❌ جنسیت باید از لیست زیر انتخاب شود", GenderEditKeyboardWithCancel())
		default:
			return editOrSend(c, messages.ErrInvalidInput, CancelKeyboard())
		}
	}

	h.redis.ClearUserState(u.TelegramID)

	return editOrSend(c, messages.UpdateSuccessful, MainMenuKeyboard())

}

func (h *Handler) HandleEditPhoto(c tele.Context, u *user.User) error {

	photos := c.Message().Photo

	if photos == nil {

		return editOrSend(c, "❌ لطفاً یک عکس ارسال کنید", CancelKeyboard())
	}

	fileID := photos.FileID

	if err := h.users.UpdateOptionalField(u, "photo", fileID); err != nil {

		return editOrSend(c, "❌ خطا در ذخیره عکس", CancelKeyboard())
	}

	h.redis.ClearUserState(u.TelegramID)

	return editOrSend(c, "✅ عکس پروفایل به‌روزرسانی شد", MainMenuKeyboard())

}

func (h *Handler) HandleEditGPS(c tele.Context, u *user.User) error {

	loc := c.Message().Location

	if loc == nil {

		return editOrSend(c, "❌ لطفاً موقعیت مکانی ارسال کنید", CancelKeyboard())
	}

	Lat := float64(loc.Lat)
	Lng := float64(loc.Lng)

	if err := h.users.UpdateGPS(u, Lat, Lng); err != nil {

		return editOrSend(c, messages.ErrSaveGPS, CancelKeyboard())
	}

	h.redis.ClearUserState(u.TelegramID)

	return editOrSend(c, messages.GPSUpdated, MainMenuKeyboard())

}

func (h *Handler) CancelHandler(c tele.Context) error {

	u, _, err := h.users.GetOrCreate(c.Sender().ID)

	if err != nil {

		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	c.Respond()

	h.redis.ClearUserState(u.TelegramID)

	return c.Edit(messages.Cancelled, MainMenuKeyboard())

}

// My Profile Actions
// ─── GPS ──────────────────────────────────────────────

func (h *Handler) ViewGPSHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "خطا"})
	}

	if u.Latitude == nil || u.Longitude == nil {
		return c.Respond(&tele.CallbackResponse{
			Text:      messages.GpsNotSet,
			ShowAlert: true,
		})
	}

	_ = c.Respond(&tele.CallbackResponse{})
	return c.Reply(&tele.Location{
		Lat: float32(*u.Latitude),
		Lng: float32(*u.Longitude),
	})
}

func (h *Handler) NoGPSHandler(c tele.Context) error {
	return c.Respond(&tele.CallbackResponse{
		Text:      messages.GpsNotRegistered,
		ShowAlert: true,
	})
}

// ─── Contacts ─────────────────────────────────────────

// func (h *Handler) ContactsHandler(c tele.Context) error {
// 	_ = c.Respond(&tele.CallbackResponse{})

// 	contacts, err := h.users.GetContacts(c.Sender().ID)
// 	if err != nil {
// 		return editOrSend(c, "❌ خطا در دریافت مخاطبین")
// 	}

// 	if len(contacts) == 0 {
// 		return editOrSend(c, "👫 هنوز مخاطبی ندارید")
// 	}

// 	text := "👫 *مخاطبین شما:*\n\n"
// 	for i, u := range contacts {
// 		age := "نامشخص"
// 		if u.Age != nil {
// 			age = fmt.Sprintf("%d سال", *u.Age)
// 		}
// 		text += fmt.Sprintf("%d. %s | %s | %s\n",
// 			i+1, u.Name, age, u.Province,
// 		)
// 	}

// 	return editOrSend(c, text, tele.ModeMarkdown)
// }

// ─── My Likes ─────────────────────────────────────────
func (h *Handler) buildLikesMessage(u *user.User, page int) (string, *tele.ReplyMarkup, error) {
	const pageSize = myLikesPageSize

	users, total, err := h.users.GetLikedBy(u.TelegramID, page, pageSize)
	if err != nil {
		return "", nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	if total == 0 {
		msg := "👥 هنوز کسی پروفایل شما را لایک ❤️ نکرده است."
		kb := MyLikesKeyboard(1, 1, u.DisableLikes)
		return msg, kb, nil
	}

	var sb strings.Builder
	sb.WriteString("👥 لیست کاربرانی که پروفایل شما را لایک ❤️ کرده اند در زیر آمده است.\n\n")

	offset := (page - 1) * pageSize
	for i, liker := range users {
		// نام
		name := liker.Name
		if name == "" {
			name = "؟"
		}

		// جنسیت
		gender := "🧑"
		if liker.Gender == user.Female {
			gender = "👩"
		}

		// لینک
		link := fmt.Sprintf("/user_%s", liker.ID)

		// سن
		age := "؟"
		if liker.Age != nil {
			age = fmt.Sprintf("%d", *liker.Age)
		}

		// استان

		Province := liker.Province

		// فاصله
		distStr := ""
		if u.Latitude != nil && u.Longitude != nil &&
			liker.Latitude != nil && liker.Longitude != nil {
			dist := calcDistance(*u.Latitude, *u.Longitude, *liker.Latitude, *liker.Longitude)
			distStr = fmt.Sprintf(" (%.1fkm 🏁)", dist)
		}

		// آخرین بازدید
		lastSeen := formatLastSeen(liker.LastSeenAt)

		sb.WriteString(fmt.Sprintf(
			"%d. %s %s %s\n%s %s%s\n%s\n〰〰〰〰〰〰〰〰〰\n",
			offset+i+1,
			gender,
			name,
			link,
			age,
			Province,
			distStr,
			lastSeen,
		))
	}

	sb.WriteString(fmt.Sprintln("\n— برای حذف دکمه لایک از پروفایلتان میتوانید این بخش را کلیک روی\nدکمه غیر فعال سازی بخش لایک غیر فعال کنید 👇"))

	kb := MyLikesKeyboard(page, totalPages, u.DisableLikes)

	return sb.String(), kb, nil
}

func (h *Handler) MyLikesHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return editOrSend(c, messages.ErrGeneric, MainMenuKeyboard())
	}

	msg, kb, err := h.buildLikesMessage(u, 1)
	if err != nil {
		return editOrSend(c, "❌ خطا در دریافت لیست", MainMenuKeyboard())
	}

	// If invoked via an inline callback, edit the existing message in place.
	if c.Callback() != nil {
		c.Respond()
		return c.Edit(msg, kb)
	}
	return editOrSend(c, msg, kb)
}

func (h *Handler) LikesPageHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	c.Respond()

	page, err := strconv.Atoi(c.Callback().Data)
	if err != nil || page < 1 {
		page = 1
	}

	msg, kb, err := h.buildLikesMessage(u, page)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در دریافت لیست"})
	}

	return c.Edit(msg, kb)
}

func (h *Handler) ToggleLikesHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	newVal := !u.DisableLikes
	u.DisableLikes = newVal

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	if err := h.db.Update(ctx, u); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ثبت تغییر"})
	}

	c.Respond()

	u.DisableLikes = newVal
	msg, kb, err := h.buildLikesMessage(u, 1)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	return c.Edit(msg, kb)
}

// ─── Silent ───────────────────────────────────────────

var ForeverSilent = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

func IsSilent(u *user.User) bool {
	if u.SilentUntil == nil {
		return false
	}
	if u.SilentUntil.Equal(ForeverSilent) {
		return true
	}
	return time.Now().Before(*u.SilentUntil)
}

func silentStatusMessage(u *user.User) string {
	if !IsSilent(u) {
		return "🔔 حالت سایلنت : غیر فعال\n\n💡 با فعال شدن حالت سایلنت، درخواست چت دریافت نخواهید کرد."
	}
	if u.SilentUntil.Equal(ForeverSilent) {
		return "🔕 حالت سایلنت : فعال (همیشه)\n\n💡 در حال حاضر درخواست چت دریافت نمی‌کنید."
	}
	return fmt.Sprintf("🔕 حالت سایلنت : فعال تا %s\n\n💡 در حال حاضر درخواست چت دریافت نمی‌کنید.",
		u.SilentUntil.Format("15:04"))
}

func (h *Handler) SilentHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	// lazy reset: اگه سایلنت موقت بود و وقتش گذشته، پاکش کن
	if u.SilentUntil != nil && !u.SilentUntil.Equal(ForeverSilent) &&
		time.Now().After(*u.SilentUntil) {
		u.SilentUntil = nil
		if err := h.users.Save(u); err != nil {
			h.log.Error("failed to clear expired silent state", zap.Error(err))
		}
	}

	if c.Callback() != nil {
		c.Respond()
		return c.Edit(silentStatusMessage(u), SilentKeyboard(u))
	}
	return editOrSend(c, silentStatusMessage(u), SilentKeyboard(u))
}

func (h *Handler) SilentForeverHandler(c tele.Context) error {
	return h.setSilent(c, &ForeverSilent)
}

func (h *Handler) SilentHourHandler(c tele.Context) error {
	t := time.Now().Add(time.Hour)
	return h.setSilent(c, &t)
}

func (h *Handler) Silent20Handler(c tele.Context) error {
	t := time.Now().Add(20 * time.Minute)
	return h.setSilent(c, &t)
}

func (h *Handler) SilentOffHandler(c tele.Context) error {
	return h.setSilent(c, nil)
}

func (h *Handler) setSilent(c tele.Context, until *time.Time) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	u.SilentUntil = until
	if err := h.users.Save(u); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در ذخیره"})
	}

	// اگه سایلنت موقت بود، lazy reset توی ChatRequestHandler انجام میشه
	c.Respond()
	return c.Edit(silentStatusMessage(u), SilentKeyboard(u))
}

// contact
const contactsPageSize = 10

// وقتی کاربر دکمه "افزودن به مخاطبین" روی پروفایل کسی میزنه
func (h *Handler) AddContactHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	contactID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	// // چک کن state فعلی نداشته باشه
	// state, _ := h.redis.GetUserState(u.TelegramID)
	// if state != "" {
	// 	return c.Respond(&tele.CallbackResponse{Text: "⚠️ ابتدا عملیات فعلی را تکمیل یا لغو کنید"})
	// }

	h.redis.SetPendingContact(u.TelegramID, contactID)
	h.redis.SetUserState(u.TelegramID, session.StateAddContactLabel)

	c.Respond()
	return editOrSend(c,
		"👤 شما در حال ذخیره کردن کاربر در لیست مخاطبین خود هستید.\n\n"+
			"در صورت تمایل برای اینکار عنوانی که بعدا بتوانید این کاربر را بیاورید ارسال کنید "+
			"یا در صورت عدم تمایل از منوی پایین روی گزینه «بازگشت ↩️» کلیک کنید.",
		CancelKeyboard(),
	)
}

// توی TextHandler وقتی state == StateAddContactLabel
func (h *Handler) HandleAddContactLabel(c tele.Context) error {
	label := strings.TrimSpace(c.Text())
	if label == "" {
		return editOrSend(c, "⚠️ لطفاً یک عنوان وارد کنید.")
	}

	contactID, err := h.redis.GetPendingContact(c.Sender().ID)
	if err != nil {
		h.redis.ClearUserState(c.Sender().ID)
		return editOrSend(c, "❌ خطا، لطفاً دوباره امتحان کنید.", MainMenuKeyboard())
	}

	if err := h.users.AddContact(c.Sender().ID, contactID, label); err != nil {
		return editOrSend(c, "❌ خطا در ذخیره‌سازی.", MainMenuKeyboard())
	}

	h.redis.ClearUserState(c.Sender().ID)
	h.redis.DelPendingContact(c.Sender().ID)

	return editOrSend(c,
		"👥 مخاطب ذخیره شد ✅\n\nتوجه: مخاطبین خود را می‌توانید از قسمت مخاطبین که در بخش پروفایل قرار دارد مشاهده کنید.\n\nخب، حالا چه کاری برات انجام بدم؟\n\nاز منوی پایین👇 انتخاب کن",
		MainMenuKeyboard(),
	)
}

// حذف از مخاطبین (از روی پروفایل)
func (h *Handler) RemoveContactHandler(c tele.Context) error {
	contactID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}

	if err := h.users.RemoveContact(c.Sender().ID, contactID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در حذف"})
	}

	c.Respond()
	// کیبورد پروفایل رو آپدیت کن (دکمه برگرده به افزودن)
	return c.Edit("با موفقیت از لیست مخاطبین حذف شد.")
}

// مشاهده لیست مخاطبین - صفحه اول.
// Triggered by an inline-keyboard callback, so we edit the existing message
// instead of sending a new one (falls back to send for non-callback callers).
func (h *Handler) MyContactsHandler(c tele.Context) error {
	if c.Callback() != nil {
		c.Respond()
		return h.editContactsPage(c, 1)
	}
	return h.sendContactsPage(c, 1)
}

// pagination
func (h *Handler) ContactsPageHandler(c tele.Context) error {
	page, err := strconv.Atoi(c.Callback().Data)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	c.Respond()
	return h.editContactsPage(c, page)
}

func (h *Handler) sendContactsPage(c tele.Context, page int) error {
	msg, kb, err := h.buildContactsPage(c.Sender().ID, page)
	if err != nil {
		return editOrSend(c, "❌ خطا در دریافت مخاطبین.")
	}
	return editOrSend(c, msg, kb)
}

func (h *Handler) editContactsPage(c tele.Context, page int) error {
	msg, kb, err := h.buildContactsPage(c.Sender().ID, page)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	return c.Edit(msg, kb)
}

func (h *Handler) buildContactsPage(ownerID int64, page int) (string, *tele.ReplyMarkup, error) {
	offset := (page - 1) * contactsPageSize
	contacts, total, err := h.users.GetContacts(ownerID, offset, contactsPageSize)
	if err != nil {
		return "", nil, err
	}

	if total == 0 {
		return "👥 لیست مخاطبین شما خالی است.", &tele.ReplyMarkup{}, nil
	}

	// Batch-load all referenced users in one query to avoid N+1.
	ids := make([]int64, 0, len(contacts))
	for _, contact := range contacts {
		ids = append(ids, contact.ContactID)
	}
	loaded, _ := h.users.BatchGetByTelegramIDs(ids)
	byID := make(map[int64]*user.User, len(loaded))
	for i := range loaded {
		byID[loaded[i].TelegramID] = &loaded[i]
	}

	totalPages := int(math.Ceil(float64(total) / float64(contactsPageSize)))
	var sb strings.Builder
	sb.Grow(256 * len(contacts))
	sb.WriteString("👥👤 لیست مخاطبین شما\n\n")

	for i, contact := range contacts {
		target := byID[contact.ContactID]
		if target == nil {
			continue
		}
		num := offset + i + 1
		gender := "👤"
		if target.Gender == user.Female {
			gender = "👩"
		}
		sb.WriteString(fmt.Sprintf("%d. %s ❓ (%s) /user_%s\n", num, gender, contact.Label, target.ID))
		if target.City != "" {
			sb.WriteString("   ")
			sb.WriteString(target.City)
			sb.WriteByte('\n')
		}
		sb.WriteString("   ")
		sb.WriteString(formatLastSeen(target.LastSeenAt))
		sb.WriteString(" ⏳\n〰〰〰〰〰〰〰〰〰〰\n\n")
	}

	sb.WriteString("➖ برای حذف کاربر روی پروفایل کاربر گزینه «حذف از مخاطبین» را بزنید.\n")
	sb.WriteString("🗑 حذف همه مخاطبین : /deleteAllContacts")

	return sb.String(), ContactsNavKeyboard(page, totalPages), nil
}

// حذف همه مخاطبین
func (h *Handler) DeleteAllContactsHandler(c tele.Context) error {
	if err := h.users.DeleteAllContacts(c.Sender().ID); err != nil {
		return editOrSend(c, "❌ خطا در حذف مخاطبین.")
	}
	return editOrSend(c, "🗑 همه مخاطبین حذف شدند.", MainMenuKeyboard())
}

// blocked
const blocksPageSize = 10

// بلاک کردن کاربر
func (h *Handler) BlockHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	targetID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	if err := h.users.BlockUser(u.TelegramID, targetID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در بلاک کردن"})
	}

	kb := &tele.ReplyMarkup{}
	ack := kb.Data("متوجه شدم", "btnBlockAck")
	kb.Inline(kb.Row(ack))

	c.Respond()
	return editOrSend(c, "🚨 کاربر بلاک شد.\n\nاین کاربر امکان ارسال درخواست چت و پیام دایرکت به شما را نخواهد داشت.", kb)
}

// تایید پیام بلاک
func (h *Handler) BlockAckHandler(c tele.Context) error {
	c.Respond()
	return c.Delete()
}

// آنبلاک کردن کاربر — answers the callback with a toast and replaces the
// profile-action keyboard so the button flips to "block" immediately.
func (h *Handler) UnblockHandler(c tele.Context) error {
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	targetID, err := strconv.ParseInt(c.Callback().Data, 10, 64)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	if err := h.users.UnblockUser(u.TelegramID, targetID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا در آنبلاک کردن"})
	}
	return c.Respond(&tele.CallbackResponse{Text: "✅ کاربر آنبلاک شد"})
}

// نمایش صفحه اول لیست بلاک‌شده‌ها
func (h *Handler) BlocksHandler(c tele.Context) error {
	if c.Callback() != nil {
		c.Respond()
		return h.editBlocksPage(c, 1)
	}
	return h.sendBlocksPage(c, 1)
}

// pagination لیست بلاک‌شده‌ها
func (h *Handler) BlocksPageHandler(c tele.Context) error {
	page, err := strconv.Atoi(c.Callback().Data)

	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	c.Respond()
	return h.editBlocksPage(c, page)
}

// حذف همه بلاک‌شده‌ها
func (h *Handler) DeleteAllBlocksHandler(c tele.Context) error {
	if err := h.users.DeleteAllBlocks(c.Sender().ID); err != nil {
		return editOrSend(c, "❌ خطا در حذف.")
	}
	return editOrSend(c, "🗑 همه بلاک‌شده‌ها حذف شدند.", MainMenuKeyboard())
}

// ارسال صفحه جدید
func (h *Handler) sendBlocksPage(c tele.Context, page int) error {
	msg, kb, err := h.buildBlocksPage(c.Sender().ID, page)
	if err != nil {
		return editOrSend(c, "❌ خطا در دریافت لیست.")
	}
	return editOrSend(c, msg, kb)
}

// ویرایش صفحه موجود
func (h *Handler) editBlocksPage(c tele.Context, page int) error {
	msg, kb, err := h.buildBlocksPage(c.Sender().ID, page)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: messages.ErrGeneric})
	}
	return c.Edit(msg, kb)
}

// ساخت محتوای صفحه
func (h *Handler) buildBlocksPage(blockerID int64, page int) (string, *tele.ReplyMarkup, error) {
	offset := (page - 1) * blocksPageSize
	blocks, total, err := h.users.GetBlocked(blockerID, offset, blocksPageSize)
	if err != nil {
		return "", nil, err
	}

	if total == 0 {
		return "🚫 لیست بلاک‌شده‌های شما خالی است.", &tele.ReplyMarkup{}, nil
	}

	ids := make([]int64, 0, len(blocks))
	for _, b := range blocks {
		ids = append(ids, b.BlockedID)
	}
	loaded, _ := h.users.BatchGetByTelegramIDs(ids)
	byID := make(map[int64]*user.User, len(loaded))
	for i := range loaded {
		byID[loaded[i].TelegramID] = &loaded[i]
	}

	totalPages := int(math.Ceil(float64(total) / float64(blocksPageSize)))
	var sb strings.Builder
	sb.Grow(192 * len(blocks))
	sb.WriteString("👥 لیست کاربران بلاک شده\n\n")

	for i, block := range blocks {
		target := byID[block.BlockedID]
		if target == nil {
			continue
		}
		num := offset + i + 1
		gender := "👤"
		if target.Gender == user.Female {
			gender = "👩"
		}
		sb.WriteString(fmt.Sprintf("%d. %s /user_%s\n", num, gender, target.ID))
		if target.City != "" {
			sb.WriteString("   📍 ")
			sb.WriteString(target.City)
			sb.WriteByte('\n')
		}
		sb.WriteString("   🕐 ")
		sb.WriteString(formatLastSeen(target.LastSeenAt))
		sb.WriteString("\n〰〰〰〰〰〰〰〰〰〰\n\n")
	}

	sb.WriteString("➖ برای آنبلاک کردن روی پروفایل کاربر گزینه «آنبلاک کردن کاربر» را بزنید.\n")
	sb.WriteString("🗑 حذف همه: /deleteAllBlocks")

	return sb.String(), BlocksNavKeyboard(page, totalPages), nil
}

// کیبورد ناوبری
func BlocksNavKeyboard(page, totalPages int) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var nav []tele.Btn

	if page > 1 {
		prev := kb.Data("⬅️ صفحه قبل", "blocksPage", fmt.Sprintf("%d", page-1))
		nav = append(nav, prev)
	}
	if page < totalPages {
		next := kb.Data("➡️ ادامه لیست", "blocksPage", fmt.Sprintf("%d", page+1))
		nav = append(nav, next)
	}

	if len(nav) > 0 {
		kb.Inline(kb.Row(nav...))
	}
	return kb
}
