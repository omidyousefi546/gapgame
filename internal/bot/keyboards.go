package bot

import (
	"GapGame/internal/service"
	"GapGame/internal/user"
	"GapGame/internal/utils"
	"fmt"
	"strconv"

	tele "gopkg.in/telebot.v3"
)

func MainMenuKeyboard() *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	kb.Reply(
		kb.Row(btnConnect),
		kb.Row(btnGame, btnSearch),
		kb.Row(btnProfile, btnInvite, btnCoins),
		kb.Row(btnHelp),
		kb.Row(btnStart),
	)
	return kb
}

func RestartKeyboard() *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	kb.Reply(kb.Row(btnStart))

	return kb

}

func CancelKeyboard() *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	kb.Inline(
		kb.Row(btnCancelEdit))

	return kb

}
func GenderKeyboard() *tele.ReplyMarkup {

	markup := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Inline(

		markup.Row(btnMale, btnFemale),
	)

	return markup

}

func GenderEditKeyboardWithCancel() *tele.ReplyMarkup {

	markup := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Inline(

		markup.Row(btnMaleEdit, btnFemaleEdit),
		markup.Row(btnCancelEdit),
	)

	return markup

}

func GenderEditKeyboard() *tele.ReplyMarkup {

	markup := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	markup.Inline(

		markup.Row(btnMaleEdit, btnFemaleEdit),
	)

	return markup

}

func ProvinceKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	var rows []tele.Row
	for i := 0; i < len(utils.IranProvinces); i += 2 {
		if i+1 < len(utils.IranProvinces) {
			rows = append(rows, kb.Row(
				kb.Text(utils.IranProvinces[i]),
				kb.Text(utils.IranProvinces[i+1]),
			))
		} else {
			rows = append(rows, kb.Row(kb.Text(utils.IranProvinces[i])))
		}
	}

	kb.Reply(rows...)
	return kb
}
func ProfileInlineKeyboard(u *user.User) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	gpsBtn := btnViewGPS
	if u.Latitude == nil {
		gpsBtn = btnNoGPS
	}
	menu.Inline(
		menu.Row(gpsBtn),
		menu.Row(btnContacts, btnMyLikes),
		menu.Row(btnBlocksList, btnSilent), // لیست بلاک‌شده‌ها
		menu.Row(btnEditProfile),
	)
	return menu
}

// GPS
func LocationKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	kb.Reply(
		kb.Row(btnSendGPS),
		kb.Row(btnBackToEditProfile),
	)

	return kb
}
func ProfileActionKeyboard(users *service.UserService, viewer *user.User, u *user.User, isOnline bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	rows := []tele.Row{}

	targetIDStr := fmt.Sprint(u.TelegramID)

	isBlocked, _ := users.IsBlocked(viewer.TelegramID, u.TelegramID)
	isContact, _ := users.IsContact(viewer.TelegramID, u.TelegramID)

	if !u.DisableLikes {
		rows = append(rows, kb.Row(
			kb.Data(fmt.Sprintf("Like ❤️ %d", u.Likes), "btnLike", targetIDStr),
		))
	}

	rows = append(rows, kb.Row(
		kb.Data("✉️ پیام دایرکت", "btnDM", targetIDStr),
		kb.Data("💬 درخواست چت", "btnChatRequest", targetIDStr),
	))

	// دکمه بلاک/آنبلاک
	var blockBtn tele.Btn
	if isBlocked {
		blockBtn = kb.Data("✅ آنبلاک کردن کاربر", "btnUnblock", targetIDStr)
	} else {
		blockBtn = kb.Data("🚫 بلاک کردن کاربر", "btnBlock", targetIDStr)
	}

	// دکمه مخاطب
	var contactBtn tele.Btn
	if isContact {
		contactBtn = kb.Data("❌ حذف از مخاطبین", "btnRemoveContact", targetIDStr)
	} else {
		contactBtn = kb.Data("➕ افزودن به مخاطبین", "btnAddContact", targetIDStr)
	}

	rows = append(rows,
		kb.Row(blockBtn, contactBtn),
		kb.Row(kb.Data("⚠️ گزارش", "btnReport", targetIDStr)),
	)

	if !isOnline {
		rows = append(rows, kb.Row(
			kb.Data("🔔 به محض آنلاین شدن اطلاع بده", "btnNotifyOnline", targetIDStr),
		))
	}

	kb.Inline(rows...)
	return kb
}

func SilentKeyboard(u *user.User) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	if IsSilent(u) {
		kb.Inline(
			kb.Row(btnSilentOff),
		)
	} else {
		kb.Inline(
			kb.Row(btnSilent20, btnSilentHour),
			kb.Row(btnSilentForever),
		)
	}
	return kb
}

// contact

func ContactsNavKeyboard(page, totalPages int) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	var rows []tele.Row
	var nav []tele.Btn

	if page > 1 {
		prev := kb.Data("⬅️ صفحه قبل", "contactsPage", fmt.Sprintf("%d", page-1))
		nav = append(nav, prev)
	}
	if page < totalPages {
		next := kb.Data("➡️ مشاهده ادامه لیست", "contactsPage", fmt.Sprintf("%d", page+1))
		nav = append(nav, next)
	}
	if len(nav) > 0 {
		rows = append(rows, kb.Row(nav...))
	}
	kb.Inline(rows...)
	return kb
}

func ChatRequestKeyboard(u *user.User) *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{}

	kb.Inline(

		kb.Row(
			kb.Data("✅ قبول", "btnAcceptChat", strconv.FormatInt(u.TelegramID, 10)), kb.Data("❌ رد", "btnRejectChat", strconv.FormatInt(u.TelegramID, 10)),
		),
	)

	return kb

}

// My profile actions
const myLikesPageSize = 10

func MyLikesKeyboard(page int, totalPages int, disableLikes bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	toggleBtn := markup.Data("🔴 غیرفعال کردن بخش لایک ❤️", "toggle_likes")
	if disableLikes {
		toggleBtn = markup.Data("🟢 فعال کردن بخش لایک ❤️", "toggle_likes")
	}
	rows = append(rows, markup.Row(toggleBtn))

	var navRow tele.Row
	if page > 1 {
		navRow = append(navRow, markup.Data("◀️ صفحه قبل", "likes_page", fmt.Sprintf("%d", page-1)))
	}
	if page < totalPages {
		navRow = append(navRow, markup.Data("صفحه بعد ▶️", "likes_page", fmt.Sprintf("%d", page+1)))
	}
	if len(navRow) > 0 {
		rows = append(rows, navRow)
	}

	markup.Inline(rows...)
	return markup
}

// search
func SearchMainKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("👥 هم سنی ها", "stype", "age"),
			kb.Data("🎯 هم استانی ها", "stype", "province"),
		),
		kb.Row(kb.Data("📍 افراد نزدیک به من", "stype", "nearby")), // جدید

		kb.Row(
			kb.Data("🚶 بدون چت ها", "stype", "nochat"),
			kb.Data("🙋 کاربران جدید ...", "stype", "new"),
		),
	)
	return kb
}

func SearchGenderKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(
			kb.Data("🧑 فقط پسر ها", "sgender", "male"),
			kb.Data("👩 فقط دختر ها", "sgender", "female"),
		),
		kb.Row(kb.Data("همه رو نشون بده", "sgender", "all")),
	)
	return kb
}

func SearchProvinceKeyboard(selected []string) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	selectedMap := map[string]bool{}
	for _, p := range selected {
		selectedMap[p] = true
	}

	rows := []tele.Row{
		kb.Row(kb.Data("➡️ مرحله بعدی", "sprov", "next")),
		kb.Row(
			kb.Data("✅ انتخاب همه", "sprov", "all"),
			kb.Data("📍 افراد نزدیک من", "sprov", "near"),
		),
	}

	for i := 0; i < len(utils.IranProvinces); i += 3 {
		end := i + 3
		if end > len(utils.IranProvinces) {
			end = len(utils.IranProvinces)
		}
		chunk := utils.IranProvinces[i:end]
		var btns []tele.Btn
		for _, p := range chunk {
			label := p
			if selectedMap[p] {
				label = "✅ " + p
			}
			btns = append(btns, kb.Data(label, "sprov", p))
		}
		rows = append(rows, kb.Row(btns...))
	}

	kb.Inline(rows...)
	return kb
}

// صفحه اول: فقط "مشاهده ادامه لیست" + "کشویی"
// صفحات بعد: "لیست قبلی" + "لیست بعدی" + "کشویی"
func SearchResultKeyboard(hasPrev, hasNext bool) *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	var rows []tele.Row

	if hasPrev || hasNext {
		var navBtns []tele.Btn
		if hasPrev {
			navBtns = append(navBtns, kb.Data("⬅️ لیست قبلی", "spage", "prev"))
		}
		if hasNext {
			navBtns = append(navBtns, kb.Data("➡️ لیست بعدی", "spage", "next"))
		}
		rows = append(rows, kb.Row(navBtns...))
	} else if hasNext {
		rows = append(rows, kb.Row(kb.Data("➡️ مشاهده ادامه لیست", "spage", "next")))
	}

	if !hasPrev && hasNext {
		rows = []tele.Row{
			kb.Row(kb.Data("➡️ مشاهده ادامه لیست", "spage", "next")),
		}
	}

	rows = append(rows, kb.Row(kb.Data("⚡ مشاهده بصورت کشویی", "spage", "swipe")))

	kb.Inline(rows...)
	return kb
}

// func SearchFilterKeyboard() *tele.ReplyMarkup {

// 	kb := &tele.ReplyMarkup{}

// 	kb.Inline(

// 		kb.Row(
// 			kb.Data("👨 مرد", "filter_gender_male"),
// 			kb.Data("👩 زن", "filter_gender_female"), kb.Data("👥 همه", "filter_gender_all"),
// 		),
// 		kb.Row(kb.Data("🟢 فقط آنلاین", "filter_online")),
// 		kb.Row(kb.Data("🔍 جستجو", "search")),
// 	)

// 	return kb

// }

func OptionalCompletionKeyboard() *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{ResizeKeyboard: true}

	kb.Inline(
		kb.Row(btnStart_optional),
		kb.Row(btnSkip_optional),
	)
	return kb
}

func EditProfileKeyboard() *tele.ReplyMarkup {

	kb := &tele.ReplyMarkup{}

	kb.Inline(

		kb.Row(btnEditName, btnEditGender),
		kb.Row(btnEditAge, btnEditCity),
		kb.Row(btnEditProvince, btnEditPhoto),
		kb.Row(btnEditGPS),
	)

	return kb

}

// chat Anonymous

// در keyboard.go اضافه کن

func ConnectMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(tele.Btn{Unique: "ctype", Text: "🎲 جستجو شانسی 🎲", Data: "random"}),
		menu.Row(
			tele.Btn{Unique: "ctype", Text: "🧑 جستجو پسر", Data: "male"},
			tele.Btn{Unique: "ctype", Text: "👩 جستجو دختر ...", Data: "female"},
		),
		menu.Row(tele.Btn{Unique: "ctype", Text: "🛰️ جستجوی اطراف", Data: "nearby"}),
	)
	return menu
}

func NearbyMenuKeyboard(hasGPS bool) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	if hasGPS {
		rows = append(rows, menu.Row(tele.Btn{Unique: "ntype", Text: "📍 افراد نزدیک من", Data: "nearby_near"}))
	}
	rows = append(rows,
		menu.Row(
			tele.Btn{Unique: "ntype", Text: "🧑 پسر باشه (💰4)", Data: "nearby_male"},
			tele.Btn{Unique: "ntype", Text: "👩 دختر باشه (💰4)", Data: "nearby_female"},
		),
		menu.Row(tele.Btn{Unique: "ntype", Text: "فرقی نمیکنه (رایگان)", Data: "nearby_all"}),
	)
	menu.Inline(rows...)
	return menu
}

func ActiveChatKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	menu.Reply(
		menu.Row(btnViewChatProfile, btnChatGame),
		menu.Row(btnEndChat),
	)
	return menu
}

func ConfirmEndChatKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			tele.Btn{Unique: "confirm_end_chat", Text: "اتمام چت ❌"},
			tele.Btn{Unique: "cancel_end_chat", Text: "ادامه چت 🗣️"},
		),
	)
	return menu
}

// SearchAgainKeyboard is shown when the 2-minute automatic search ends with
// no match. The previously selected filter is carried in the callback data so
// «🔁 جستجوی مجدد» re-runs the search with the exact same filters.
func SearchAgainKeyboard(filter string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(tele.Btn{Unique: "search_again", Text: "🔁 جستجوی مجدد", Data: filter}),
	)
	return menu
}

func ChatGameMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(btnChatGameRPS, btnChatGameWord),
		menu.Row(btnChatGameDooz4, btnChatGameDoozClass),
		menu.Row(btnChatGameDareTruth),
	)
	return menu
}

func ChatGameRequestKeyboard(gameType string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			tele.Btn{Unique: "cgame_acc", Text: "✅ قبول", Data: gameType},
			tele.Btn{Unique: "cgame_rej", Text: "❌ رد", Data: gameType},
		),
	)
	return menu
}

// Games

func GameMenuKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}
	kb.Inline(
		kb.Row(btnGameDooz4Normal),
		kb.Row(btnGameDooz4Gravity),
		kb.Row(btnGameDoozClassic),
		kb.Row(btnGameDareAndTruth),
	)
	return kb
}

func InGameMenuKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	kb.Reply(
		kb.Row(btnFinishGame),
	)
	return kb
}

func ConfirmFinishGameMenuKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	kb.Inline(
		kb.Row(btnConfirmFinishGame),
		kb.Row(btnDeclineFinishGame),
	)
	return kb
}

func AfterGameMenuKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	kb.Inline(
		kb.Row(btnRepeatGame, btnNewGame),
		kb.Row(btnCancelGame),
	)
	return kb
}

// ///////////////////////////////////////////// game keyboard
func boardDooz4Keyboard(Board *[7][7]int, unique string) *tele.ReplyMarkup {

	m := &tele.ReplyMarkup{}
	row := [][]tele.Btn{}

	for r := 0; r < 7; r++ {

		col := []tele.Btn{}

		for c := 0; c < 7; c++ {

			emoji := "⚪"

			if Board[c][r] == 1 {
				emoji = "🔴"
			}

			if Board[c][r] == 2 {
				emoji = "🔵"
			}

			btn := m.Data(emoji, unique, fmt.Sprintf("%d-%d", r, c))

			col = append(col, btn)
		}

		row = append(row, m.Row(col...))
	}
	m.Inline(row[0],
		row[1],
		row[2],
		row[3],
		row[4],
		row[5],
		row[6],
	)

	return m
}

func boardDooz4KeyboardDisabled(Board *[7][7]int) *tele.ReplyMarkup {

	m := &tele.ReplyMarkup{}
	row := [][]tele.Btn{}

	for r := 0; r < 7; r++ {

		col := []tele.Btn{}

		for c := 0; c < 7; c++ {

			emoji := "⚪"

			if Board[c][r] == 1 {
				emoji = "🔴"
			}

			if Board[c][r] == 2 {
				emoji = "🔵"
			}

			btn := m.Data(emoji, "dooz4_disabled", "x")

			col = append(col, btn)
		}

		row = append(row, m.Row(col...))
	}
	m.Inline(row[0],
		row[1],
		row[2],
		row[3],
		row[4],
		row[5],
		row[6],
	)

	return m
}

func boardDoozClassicKeyboard(Board *[3][3]int) *tele.ReplyMarkup {

	m := &tele.ReplyMarkup{}
	row := [][]tele.Btn{}

	for r := 0; r < 3; r++ {

		col := []tele.Btn{}

		for c := 0; c < 3; c++ {

			emoji := "⚪"

			if Board[c][r] == 1 {
				emoji = "🔴"
			}

			if Board[c][r] == 2 {
				emoji = "🔵"
			}

			btn := m.Data(emoji, "game_dooz_classic", fmt.Sprintf("%d-%d", r, c))

			col = append(col, btn)
		}

		row = append(row, m.Row(col...))
	}
	m.Inline(row[0],
		row[1],
		row[2],
	)

	return m
}

func boardDoozClassicKeyboardDisabled(Board *[3][3]int) *tele.ReplyMarkup {

	m := &tele.ReplyMarkup{}
	row := [][]tele.Btn{}

	for r := 0; r < 3; r++ {

		col := []tele.Btn{}

		for c := 0; c < 3; c++ {

			emoji := "⚪"

			if Board[c][r] == 1 {
				emoji = "🔴"
			}

			if Board[c][r] == 2 {
				emoji = "🔵"
			}

			btn := m.Data(emoji, "doozClassic_disabled", "x")

			col = append(col, btn)
		}

		row = append(row, m.Row(col...))
	}
	m.Inline(row[0],
		row[1],
		row[2],
	)

	return m
}

func boardDareAndTruthKeyboard() *tele.ReplyMarkup {

	m := &tele.ReplyMarkup{ResizeKeyboard: true}
	// NOTE: Unique must match the callback registered for "\fgame_dare_and_truth".
	// Previously this was "dare_and_truth_move" which caused the buttons to be unhandled.
	btnTruth := m.Data("حقیقت", "game_dare_and_truth", "حقیقت")
	btnDare := m.Data("جرات", "game_dare_and_truth", "جرات")
	btnTruth18 := m.Data("حقیقت+۱۸", "game_dare_and_truth", "حقیقت+18")
	btnDare18 := m.Data("جرات+۱۸", "game_dare_and_truth", "جرات+18")
	btnChance := m.Data("شانسی", "game_dare_and_truth", "شانسی")
	m.Inline(
		m.Row(btnTruth), m.Row(btnDare),
		m.Row(btnTruth18), m.Row(btnDare18),
		m.Row(btnChance),
	)
	return m

}
