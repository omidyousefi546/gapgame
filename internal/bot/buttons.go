package bot

import tele "gopkg.in/telebot.v3"

// Main menu buttons

var (
	btnConnect = tele.Btn{Text: "🔗 به یه ناشناس وصلم کن!"}

	btnGame = tele.Btn{Text: "بازی با دوستان"}

	btnNearby = tele.Btn{Text: "📍 افراد نزدیک"}

	btnSearch = tele.Btn{Text: "🔍 جستجوی کاربران"}

	btnProfile = tele.Btn{Text: "پروفایل"}

	btnHelp = tele.Btn{Text: "📖 راهنما"}

	btnCoins = tele.Btn{Text: "💰 سکه"}

	btnInvite = tele.Btn{Text: "🤝 معرفی به دوستان (سکه رایگان)"}

	btnStart = tele.Btn{Text: "شروع مجدد"}
)

// Gender selection

var (
	btnMale = tele.Btn{Unique: "gender_male", Text: "👨 مرد"}

	btnFemale = tele.Btn{Unique: "gender_female", Text: "👩 زن"}
)
var (
	btnMaleEdit = tele.Btn{Unique: "set_gender_male", Text: "👨 مرد"}

	btnFemaleEdit = tele.Btn{Unique: "set_gender_female", Text: "👩 زن"}
)

// GPS

var (
	btnSendGPS           = tele.Btn{Text: "📍 ارسال موقعیت جی پی اس", Location: true}
	btnBackToEditProfile = tele.Btn{Text: "⬅️ بازگشت"}
)

// Search type buttons

var (
	btnSearchAge = tele.Btn{Unique: "search_age", Text: "🎂 جستجو بر اساس سن"}

	btnSearchProvince = tele.Btn{Unique: "search_province", Text: "📍 جستجو بر اساس استان"}

	btnSearchNoChat = tele.Btn{Unique: "search_nochat", Text: "💬 افرادی که چت نکردم"}

	btnSearchNew = tele.Btn{Unique: "search_new", Text: "✨ کاربران جدید"}

	btnSearchNext = tele.Btn{Unique: "search_next", Text: "➡️ بعدی"}

	btnSearchSwipe = tele.Btn{Unique: "search_swipe", Text: "👆 نمایش کشویی"}
)

// Gender filter in search

var (
	btnGenderMale = tele.Btn{Unique: "filter_male", Text: "👨 مرد"}

	btnGenderFemale = tele.Btn{Unique: "filter_female", Text: "👩 زن"}

	btnGenderBoth = tele.Btn{Unique: "filter_both", Text: "👥 هر دو"}
)

// contact
var (
	// /btnContacts    = profileMenu.Data("👥 مخاطبین", "btnMyContacts")
	btnContactsPage  = profileMenu.Data("➡️ مشاهده ادامه لیست", "btnContactsPage")
	btnAddContact    = profileMenu.Data("➕ افزودن به مخاطبین", "btnAddContact")
	btnRemoveContact = profileMenu.Data("❌ حذف از مخاطبین", "btnRemoveContact")
)

// My Profile action buttons

var (
	profileMenu = &tele.ReplyMarkup{}

	btnViewGPS     = profileMenu.Data("📍 مشاهده موقعیت GPS", "view_gps")
	btnNoGPS       = profileMenu.Data("📍 موقعیت GPS ثبت نشده", "no_gps")
	btnContacts    = profileMenu.Data("👫 مخاطبین", "btnMyContacts")
	btnMyLikes     = profileMenu.Data("❤️ لایک های من", "my_likes")
	btnBlocksList  = profileMenu.Data("🚫 بلاک شده ها", "btnBlocksList") // پروفایل خودت
	btnSilent      = profileMenu.Data("🔔 سایلنت", "silent")
	btnEditProfile = profileMenu.Data("✏️ ویرایش اطلاعات پروفایل", "edit_profile")
)

var (
	// btnMyLikes     = tele.Btn{Unique: "my_likes"}
	btnLikesPage   = tele.Btn{Unique: "likes_page"}
	btnToggleLikes = tele.Btn{Unique: "toggle_likes"}
)

var (
	btnSilentForever = profileMenu.Data("🔕 همیشه سایلنت", "btnSilentForever")
	btnSilentHour    = profileMenu.Data("🔕 سایلنت تا یک ساعت", "btnSilentHour")
	btnSilent20      = profileMenu.Data("🔕 سایلنت تا ۲۰ دقیقه", "btnSilent20")
	btnSilentOff     = profileMenu.Data("🔔 غیرفعال کردن سایلنت", "btnSilentOff")
)

var (
	btnBlocks     = tele.Btn{Unique: "btnBlock"} // بلاک کردن دیگران
	btnBlockAck   = tele.Btn{Unique: "btnBlockAck"}
	btnUnblock    = tele.Btn{Unique: "btnUnblock"}
	btnBlocksPage = tele.Btn{Unique: "blocksPage"} // navigation
)

// Profile action buttons

var (
	btnLike        = tele.Btn{Unique: "btnLike"}
	btnDM          = tele.Btn{Unique: "btnDM"}
	btnChatRequest = tele.Btn{Unique: "btnChatRequest"}
	//btnAddContact   = tele.Btn{Unique: "btnAddContact"}
	// btnBlock        = tele.Btn{Unique: "btnBlock"}
	btnReport       = tele.Btn{Unique: "btnReport"}
	btnNotifyOnline = tele.Btn{Unique: "btnNotifyOnline"}
)

// Edit profile buttons

var (
	btnEditName = tele.Btn{Unique: "edit_name", Text: "نام"}

	btnEditGender = tele.Btn{Unique: "edit_gender", Text: "جنسیت"}

	btnEditAge = tele.Btn{Unique: "edit_age", Text: "سن"}

	btnEditCity = tele.Btn{Unique: "edit_city", Text: "شهر"}

	btnEditProvince = tele.Btn{Unique: "edit_province", Text: "استان"}

	btnEditPhoto = tele.Btn{Unique: "edit_photo", Text: "عکس پروفایل"}

	btnEditGPS = tele.Btn{Unique: "edit_gps", Text: "موقعیت مکانی"}

	btnCancelEdit = tele.Btn{Unique: "cancel_edit", Text: "❌ لغو"}
)

// Coins buttons

var (
	btnBuyCoins      = tele.InlineButton{Unique: "btnBuyCoins"}
	btnInviteFriends = tele.InlineButton{Unique: "btnInviteFriends"}
)

//

var (
	btnStart_optional = tele.Btn{Unique: "start_optional", Text: "✅ الان تکمیل کنم"}

	btnSkip_optional = tele.Btn{Unique: "skip_optional", Text: "⏭ بعدا"}
)

// chat Anonymous

var (
	btnConnectRandom = tele.Btn{Unique: "ctype", Data: "random"}
	btnConnectMale   = tele.Btn{Unique: "ctype", Data: "male"}
	btnConnectFemale = tele.Btn{Unique: "ctype", Data: "female"}
	btnConnectNearby = tele.Btn{Unique: "ctype", Data: "nearby"}

	btnNearbyNear   = tele.Btn{Unique: "ntype", Data: "nearby_near"}
	btnNearbyMale   = tele.Btn{Unique: "ntype", Data: "nearby_male"}
	btnNearbyFemale = tele.Btn{Unique: "ntype", Data: "nearby_female"}
	btnNearbyAll    = tele.Btn{Unique: "ntype", Data: "nearby_all"}

	btnViewChatProfile = tele.Btn{Text: "👥•• مشاهده پروفایل این مخاطب"}
	btnEndChat         = tele.Btn{Text: "پایان چت"}
	btnConfirmEndChat  = tele.Btn{Unique: "confirm_end_chat"}
	btnCancelEndChat   = tele.Btn{Unique: "cancel_end_chat"}
	btnCancelQueue     = tele.Btn{Unique: "cancel_queue"}
)

// Game menu (inline keyboard)
var (
	btnGameDooz4Normal  = tele.Btn{Unique: "game_select", Text: "بازی دوز ۴ عادی", Data: "game_dooz4_normal"}
	btnGameDooz4Gravity = tele.Btn{Unique: "game_select", Text: "بازی دوز ۴ جاذبه", Data: "game_dooz4_gravity"}
	btnGameDoozClassic  = tele.Btn{Unique: "game_select", Text: "بازی دوز کلاسیک", Data: "game_dooz_classic"}
	btnGameDareAndTruth = tele.Btn{Unique: "game_select", Text: "جرات حقیقت", Data: "game_dare_and_truth"}
)

// In Game menu (reply keyboard)

var btnFinishGame = tele.Btn{Text: "اتمام بازی"}

// Confirm Finish menu (reply keyboard)

var (
	btnConfirmFinishGame = tele.Btn{Unique: "confirmFinishGame", Text: "تایید"}
	btnDeclineFinishGame = tele.Btn{Unique: "declineFinishGame", Text: "لغو"}
)

var (
	btnRepeatGame = tele.Btn{Unique: "repeatGame", Text: "بازی مجدد"}
	btnNewGame    = tele.Btn{Unique: "newGame", Text: "انتخاب بازی"}
	btnCancelGame = tele.Btn{Unique: "cancelGame", Text: "اتمام بازی"}
)

// Keyboards

func genderFilterKeyboard() *tele.ReplyMarkup {

	markup := &tele.ReplyMarkup{}

	markup.Inline(

		markup.Row(btnGenderMale, btnGenderFemale),
		markup.Row(btnGenderBoth),
	)

	return markup

}

func cancelEditKeyboard() *tele.ReplyMarkup {

	markup := &tele.ReplyMarkup{}

	markup.Inline(markup.Row(btnCancelEdit))

	return markup

}

// func coinsKeyboard() *tele.ReplyMarkup {

// 	markup := &tele.ReplyMarkup{}

// 	markup.Inline(

// 		markup.Row(btnBuyCoins),
// 		markup.Row(btnInviteFriends),
// 		markup.Row(btnStart),
// 	)

// 	return markup

// }
