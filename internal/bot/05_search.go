package bot

import (
	"fmt"
	"strings"

	"GapGame/internal/session"
	"GapGame/internal/user"
	"GapGame/internal/utils"

	ptime "github.com/yaa110/go-persian-calendar"

	tele "gopkg.in/telebot.v3"
)

var searchTypeLabels = map[string]string{
	"age":      "هم سنی ها 👥",
	"province": "هم استانی ها 🎯",
	"advanced": "جستجوی پیشرفته 🔍",
	"nochat":   "بدون چت ها 🚶",
	"new":      "کاربران جدید 🙋",

	"nearby": "افراد نزدیک 📍", // جدید
}

var searchTypeHeaders = map[string]string{
	"age":      "👥 لیست افراد هم سن شما که در 1 روز اخیر آنلاین بوده اند",
	"province": "🎯 لیست افراد هم استانی شما که در 1 روز اخیر آنلاین بوده اند",
	"advanced": "🔍 نتایج جستجوی پیشرفته",
	"nochat":   "🚶 لیست افرادی که چت نداشته اند",
	"new":      "🙋 لیست کاربران جدید",
	"nearby":   "📍 لیست افراد نزدیک به شما", // جدید
}

const searchPageSize = 10

// ─── مرحله ۱ ─────────────────────────────────────────────────

func (h *Handler) SearchHandler(c tele.Context) error {

	return c.Send("🔍 جستجوی کاربران\n\n👆 چه کسایی رو نشونت بدم؟ انتخاب کن", SearchMainKeyboard())

}

// ─── مرحله ۲ ─────────────────────────────────────────────────
// callback unique: "stype" | data: "age" | "province" | "advanced" | "nochat" | "new"
func (h *Handler) SearchTypeHandler(c tele.Context) error {
	c.Respond()

	searchType := c.Callback().Data
	label := searchTypeLabels[searchType]

	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا در دریافت اطلاعات کاربر")
	}

	// بررسی GPS برای nearby
	if searchType == "nearby" && u.Latitude == nil {
		return c.Send("❌ برای جست‌وجوی افراد نزدیک باید موقعیت GPS خود را ثبت کنید.")
	}

	state := &session.SearchState{Type: searchType, Offset: 0}

	// ذخیره موقعیت کاربر در state
	if searchType == "nearby" {
		state.NearbyLat = u.Latitude
		state.NearbyLng = u.Longitude
	}

	h.redis.SetSearchState(u.TelegramID, state)

	return c.Send(
		fmt.Sprintf("👇 چه کسایی رو از بین %s نشونت بدم؟ انتخاب کن", label),
		SearchGenderKeyboard(),
	)
}

// ─── مرحله ۳: انتخاب جنسیت ──────────────────────────────────

// callback unique: "sgender" | data: "male" | "female" | "all"
func (h *Handler) SearchGenderHandler(c tele.Context) error {

	c.Respond()

	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا")
	}

	state, _ := h.redis.GetSearchState(u.TelegramID)
	if state == nil {
		state = &session.SearchState{Type: "age"}
	}

	switch c.Callback().Data {
	case "male":
		state.Gender = string(user.Male)
	case "female":
		state.Gender = string(user.Female)
	default:
		state.Gender = ""
	}
	state.Offset = 0
	h.redis.SetSearchState(u.TelegramID, state)

	// پیشرفته → انتخاب استان
	if state.Type == "advanced" {
		return c.Edit(
			buildProvinceMessage(state),
			SearchProvinceKeyboard(state.Provinces),
		)
	}

	// بقیه → مستقیم نتایج
	return h.sendSearchResults(c, u, state, 0)
}

// ─── مرحله ۴ (فقط advanced): toggle استان ───────────────────

// callback unique: "sprov" | data: نام استان | "all" | "near" | "next"
func (h *Handler) SearchProvinceHandler(c tele.Context) error {
	c.Respond()

	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Send("❌ خطا")
	}

	state, _ := h.redis.GetSearchState(u.TelegramID)
	if state == nil {
		state = &session.SearchState{}
	}

	data := c.Callback().Data

	if data == "next" {
		return h.sendSearchResults(c, u, state, 0)
	}

	switch data {
	case "all":
		state.Provinces = nil
	case "near":
		state.Provinces = []string{"__near__"}
	default:
		state.Provinces = toggleProvince(state.Provinces, data)
	}

	h.redis.SetSearchState(u.TelegramID, state)

	return c.Edit(buildProvinceMessage(state), SearchProvinceKeyboard(state.Provinces))
}

func buildProvinceMessage(state *session.SearchState) string {
	selectedStr := "[]"
	if len(state.Provinces) > 0 {
		if state.Provinces[0] == "__near__" {
			selectedStr = "[افراد نزدیک من]"
		} else {
			selectedStr = "[" + strings.Join(state.Provinces, "، ") + "]"
		}
	}
	return fmt.Sprintf(
		"👫 جنسیت : [%s]\n\n📍 استان های انتخاب شده : %s\n\nاستان های مورد نظرتو انتخاب کن و در آخر گزینه «➡️ مرحله بعدی» رو بزن 👇",
		genderLabel(state.Gender), selectedStr,
	)
}

// ─── نمایش نتایج (لیست ۱۰ تایی) ─────────────────────────────

func (h *Handler) sendSearchResults(c tele.Context, u *user.User, state *session.SearchState, page int) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	filter := buildFilter(u, state)
	filter.ExcludeUserID = u.TelegramID
	filter.Offset = page * searchPageSize
	filter.Limit = searchPageSize

	results, err := h.db.SearchUsers(ctx, filter)
	if err != nil {
		return c.Send("❌ خطا در جستجو")
	}

	if len(results) == 0 {
		if page == 0 {
			return c.Send("😔 کاربری با این مشخصات پیدا نشد.\nفیلترها را تغییر دهید یا بعداً دوباره امتحان کنید.")
		}
		return c.Respond(&tele.CallbackResponse{Text: "صفحه‌ای دیگری وجود ندارد"})
	}

	state.Offset = page
	h.redis.SetSearchState(u.TelegramID, state)

	header := searchTypeHeaders[state.Type]

	jalaliNow := formatJalaliNow() // جایگزین با تبدیل واقعی جلالی

	var sb strings.Builder
	sb.WriteString(header + "\n\n")

	startIndex := page * searchPageSize
	for i, target := range results {
		sb.WriteString(h.formatUserRow(startIndex+i+1, &target, u))
		sb.WriteString("\n〰〰〰〰〰〰〰〰〰〰\n\n")
	}
	sb.WriteString(fmt.Sprintf("\nجستجو شده در %s", jalaliNow))

	isFirstPage := page == 0
	hasMore := len(results) == searchPageSize

	var kb *tele.ReplyMarkup
	if isFirstPage {
		if hasMore {
			kb = SearchResultKeyboard(false, true)
		} else {
			kb = SearchResultKeyboard(false, false)
		}
	} else {
		kb = SearchResultKeyboard(true, hasMore)
	}

	if c.Callback() != nil {
		return c.Edit(sb.String(), kb)
	}
	return c.Send(sb.String(), kb)
}

func formatJalaliNow() string {
	now := ptime.Now()
	return fmt.Sprintf("%d/%02d/%02d %02d:%02d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
	)
}

func (h *Handler) formatUserRow(index int, u *user.User, MyUser *user.User) string {
	genderIcon := "👦"
	if u.Gender == user.Female {
		genderIcon = "👧"
	}

	nameOrQuestion := "❓"
	if u.Name != "" {
		nameOrQuestion = u.Name
	}

	location := u.Province
	if u.City != "" {
		location = fmt.Sprintf("%s (%s)", u.Province, u.City)
	}

	onlineStatus := fmt.Sprintf("⏳ %s قبل آنلاین بوده", formatLastSeen(u.LastSeenAt))

	distance := "-"

	if MyUser.Latitude != nil && u.Latitude != nil {
		d := calcDistance(*MyUser.Latitude, *MyUser.Longitude, *u.Latitude, *u.Longitude)

		distance = fmt.Sprintf("%.1f", d)
	}

	return fmt.Sprintf(
		"%d. %s %s /user_%s\n%d %s 🏁 (%s km)\n%s",
		index, genderIcon, nameOrQuestion, u.ID,
		u.SafeAge(), location, distance,
		onlineStatus,
	)
}

// ─── pagination handlers ──────────────────────────────────────

func (h *Handler) SearchNextPageHandler(c tele.Context) error {
	c.Respond()
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}
	state, _ := h.redis.GetSearchState(u.TelegramID)
	if state == nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}
	return h.sendSearchResults(c, u, state, state.Offset+1)
}

func (h *Handler) SearchPrevPageHandler(c tele.Context) error {
	c.Respond()
	u, _, err := h.users.GetOrCreate(c.Sender().ID)
	if err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "❌ خطا"})
	}
	state, _ := h.redis.GetSearchState(u.TelegramID)
	if state == nil || state.Offset <= 0 {
		return c.Respond(&tele.CallbackResponse{Text: "اول لیست هستی"})
	}
	return h.sendSearchResults(c, u, state, state.Offset-1)
}

func (h *Handler) SearchSwipeHandler(c tele.Context) error {
	c.Respond()
	// TODO: پیاده‌سازی حالت کشویی
	return c.Send("⚡ حالت کشویی به زودی...")
}

// ─── helpers ─────────────────────────────────────────────────

func buildFilter(u *user.User, state *session.SearchState) user.SearchFilter {
	f := user.SearchFilter{}
	if state == nil {
		return f
	}
	f.Gender = state.Gender
	f.Type = state.Type

	switch state.Type {
	case "age":
		age := u.SafeAge()
		f.MinAge = age - 3
		f.MaxAge = age + 3
	case "province":
		f.Provinces = []string{u.Province}
	case "advanced":
		if len(state.Provinces) > 0 && state.Provinces[0] != "__near__" {
			f.Provinces = state.Provinces
		}
	case "nochat":
		f.NoChat = true
	case "new":
		f.NewUsers = true
	case "nearby": // جدید
		if state.NearbyLat != nil {
			f.NearbyLat = state.NearbyLat
			f.NearbyLng = state.NearbyLng
			f.RadiusKM = 30 // شعاع پیش‌فرض ۵۰ کیلومتر
		}
	}
	return f
}

func toggleProvince(list []string, prov string) []string {
	for i, p := range list {
		if p == prov {
			return append(list[:i], list[i+1:]...)
		}
	}
	return append(list, prov)
}

func genderLabel(g string) string {
	switch g {
	case string(user.Male):
		return "پسر"
	case string(user.Female):
		return "دختر"
	default:
		return "همه"
	}
}

func (h *Handler) SearchPageHandler(c tele.Context) error {
	c.Respond()

	data := c.Callback().Data

	switch data {
	case "next":
		return h.SearchNextPageHandler(c)
	case "prev":
		return h.SearchPrevPageHandler(c)
	case "swipe":
		return h.SearchSwipeHandler(c)
	default:
		return c.Respond(&tele.CallbackResponse{Text: "❌ دستور نامعتبر"})
	}
}
