package telegram

// Persian message templates
const (
	// Welcome and registration
	MsgWelcome           = "سلام قهرمان! به دهکده دوستی خوش اومدی. 🎮 برای شروع ماجراجویی، اول جنسیتت رو مشخص کن:"
	MsgWelcomeBack       = "حوصِلت سر رفته و تنهایی؟😔🥀\n\n🏘 میخوای با هم شهریت چت کنی؟ 😍\n\n📈 دوست داری با هم سنیت یه گپی بزنی؟ ☕️\n\n👩❤️👨 میخوای جنسیت طرف مقابلتو خودت انتخاب کنی؟🤩\n\n⏳ پس منتظر چی هستی همین الان انتخاب کن و چتو شروع کن! 😍👇"
	MsgRegisterStart     = "خوشبختم! توی بازی چی صدات کنیم؟ (یک اسم کوتاه و خفن بنویس)"
	MsgRegisterName      = "✅ عالی! حالا جنسیتت رو انتخاب کن:"
	MsgRegisterGender    = "✅ خوبه! سنت چنده؟ (عدد وارد کن)"
	MsgRegisterAge       = "✅ باشه! شهرت کجاست؟"
	MsgRegisterCity      = "✅ تمام! می‌خوای عکس پروفایل هم بذاری؟\n\n(اختیاری - می‌تونی بعداً هم اضافه کنی)"
	MsgRegisterComplete  = "ثبتنامت تکمیل شد! 🎉 به عنوان هدیه ورود، ۱۰۰ سکه به کیفت اضافه شد. حالا بزن بریم!"
	MsgRegisterCancelled = "❌ ثبت نام لغو شد."

	// Main menu
	MsgMainMenu  = "حوصِلت سر رفته و تنهایی؟😔🥀\n\n🏘 میخوای با هم شهریت چت کنی؟ 😍\n\n📈 دوست داری با هم سنیت یه گپی بزنی؟ ☕️\n\n👩❤️👨 میخوای جنسیت طرف مقابلتو خودت انتخاب کنی؟🤩\n\n⏳ پس منتظر چی هستی همین الان انتخاب کن و چتو شروع کن! 😍👇"
	MsgGamesMenu = "🎮 بخش بازی‌ها:\n\nبازی مورد نظرت رو انتخاب کن:"
	MsgHelp      = "🤖 راهنمای ربات:\n\n🔍 جستجو: برای پیدا کردن یک نفر برای چت\n👤 پروفایل: مشاهده و ویرایش پروفایل\n💰 سکه‌ها: مشاهده موجودی و تاریخچه\n👥 دوستان: مدیریت دوستان\n🎮 بازی‌ها: بازی‌های گروهی با کاربران\n\nبرای شروع، از دکمه‌های زیر استفاده کن!"

	// Matchmaking
	MsgSearchStart        = "🔍 جستجو شروع شد!\n\n💰 هزینه: %d سکه\n\nداریم دنبال یک نفر مناسب برات می‌گردیم..."
	MsgSearchFilters      = "🎯 فیلترهای جستجو رو انتخاب کن:"
	MsgSelectGender       = "❓ دوست داری با کی چت کنی؟"
	MsgSearchGenderFilter = "جنسیت مورد نظر:"
	MsgSearchAgeFilter    = "محدوده سنی رو وارد کن (مثال: 20-30) یا بزن رد شو:"
	MsgSearchCityFilter   = "شهر مورد نظر رو بنویس یا بزن رد شو:"
	MsgMatchFound         = "✅ پیدا شد!\n\nیک نفر پیدا کردیم! می‌تونی شروع به چت کنی.\n\n⏰ مدت زمان: %d دقیقه"
	MsgMatchTimeout       = "⏰ زمان تموم شد!\n\n💰 بازگشت: %d سکه (نصف هزینه)\n\nمتأسفانه کسی پیدا نشد."
	MsgSearchCancelled    = "❌ جستجو لغو شد.\n\n💰 بازگشت: %d سکه"
	MsgMatchEnded         = "👋 چت تموم شد!\n\nامیدواریم لذت برده باشی."
	MsgAlreadySearching   = "⚠️ شما الان در حال جستجو هستید!"
	MsgAlreadyInMatch     = "⚠️ شما الان در یک چت فعال هستید!"
	MsgSelectSearchMode   = "🔍 چطوری میخوای جستجو کنی؟"
	MsgSelectFilter       = "📂 فیلتر مورد نظرت رو انتخاب کن:"

	// Chat
	MsgChatActive      = "💬 چت فعال است!\n\nمی‌تونی پیام بفرستی. برای پایان دادن از دکمه زیر استفاده کن."
	MsgChatCostWarning = "⚠️ چت تموم شده! هر پیام %d سکه هزینه داره.\n\nمطمئنی می‌خوای پیام بفرستی؟"
	MsgMessageSent     = "✅ پیام ارسال شد! (-💰 %d سکه)"
	MsgPartnerLeft     = "👋 طرف مقابل چت رو ترک کرد."
	MsgNoActiveChat    = "⚠️ شما در چت فعالی نیستید!"

	// Coins
	MsgBalance            = "💰 موجودی شما: %d سکه"
	MsgInsufficientCoins  = "❌ سکه کافی نداری!\n\n💰 موجودی: %d\n💰 نیاز: %d"
	MsgTransactionHistory = "📊 تاریخچه تراکنش‌ها:\n\n%s"
	MsgNoTransactions     = "📊 هنوز تراکنشی نداری!"

	// Friends
	MsgFriendRequestSent     = "✅ درخواست دوستی ارسال شد!"
	MsgFriendRequestReceived = "👥 درخواست دوستی جدید از %s"
	MsgFriendRequestAccepted = "✅ درخواست دوستی پذیرفته شد!"
	MsgFriendRequestRejected = "❌ درخواست دوستی رد شد."
	MsgFriendRemoved         = "👋 از لیست دوستان حذف شد."
	MsgFriendsList           = "👥 لیست دوستان:\n\n%s"
	MsgNoFriends             = "👥 هنوز دوستی نداری!"
	MsgAlreadyFriends        = "⚠️ شما قبلاً دوست هستید!"

	// Errors
	MsgErrorGeneric       = "❌ خطایی رخ داد! لطفاً دوباره تلاش کن."
	MsgErrorInvalidInput  = "❌ ورودی نامعتبر! دوباره تلاش کن."
	MsgErrorInvalidAge    = "❌ سن باید بین 13 تا 100 باشه!"
	MsgErrorInvalidPhoto  = "❌ فایل نامعتبر! فقط عکس (JPG, PNG) با حداکثر 5MB."
	MsgErrorNotRegistered = "⚠️ اول باید ثبت نام کنی!"
	MsgErrorUnauthorized  = "🚫 شما دسترسی به این بخش رو نداری!"

	// General
	MsgCancel     = "❌ لغو شد."
	MsgConfirm    = "✅ تأیید می‌کنی؟"
	MsgProcessing = "⏳ در حال پردازش..."
	MsgSuccess    = "✅ با موفقیت انجام شد!"

	// Purchase
	MsgCoinPurchasePlans = `💎 لیست پکیج‌های افزایش سکه:

▫️ ۲۰۰ سکه ⬅️ ۲۰،۰۰۰ تومان
▫️ ۵۰۰ سکه ⬅️ ۴۵،۰۰۰ تومان (۱۰٪ تخفیف)
▫️ ۱۰۰۰ سکه ⬅️ ۸۰،۰۰۰ تومان (۲۰٪ تخفیف)
▫️ ۵۰۰۰ سکه ⬅️ ۳۵۰،۰۰۰ تومان (۳۰٪ تخفیف ویژه)

💳 اطلاعات واریز:
۶۲۱۹۸۶۱۸۲۰۱۷۴۹۳۵
مهدی محمدی - بلو بانک

⚠️ بعد از واریز مبلغ، حتماً گزینه «واریز کردم» را انتخاب کرده و عکس رسید تراکنش را ارسال نمایید.`
	MsgRequestReceipt  = "📸 لطفاً عکس رسید یا اسکرین‌شات تراکنش خود را ارسال کنید."
	MsgPurchasePending = "✅ درخواست شما ثبت شد. پس از بررسی و تأیید واریز، سکه‌ها به حساب شما اضافه خواهد شد. از شکیبایی شما سپاسگزاریم."

	// Village
	MsgVillageWelcome       = "🏘 به سیستم دهکده خوش اومدی!\n\nدر دهکده می‌تونی با دوستانت چت کنی، بازی برگزار کنی و در رنکینگ کشوری با دهکده‌های دیگه رقابت کنی.\n\nهنوز عضو هیچ دهکده‌ای نیستی. می‌خوای دهکده خودت رو بسازی؟"
	MsgVillageInfo          = "🏡 دهکده: %s\n\n📊 سطح: %d\n✨ امتیاز (XP): %d\n🔝 رتبه: %d\n👥 اعضا: %d/50\n\n📜 %s"
	MsgVillageCreateName    = "📝 لطفا نام دهکده خود را ارسال کنید (حداکثر ۳۰ کاراکتر):"
	MsgVillageCreateDesc    = "📜 حالا یک توضیح کوتاه برای دهکده خود بنویسید:"
	MsgVillageCreateSuccess = "✅ دهکده «%s» با موفقیت ساخته شد!"
	MsgVillageLeaveSuccess  = "✅ شما با موفقیت از دهکده خارج شدید."
	MsgVillageInviteText    = "🎁 دعوت‌نامه دهکده «%s»\n\nبا کلیک بر روی لینک زیر می‌تونی به دهکده ما بیای:\n%s"
	MsgVillageJoinSuccess   = "✅ ایول! تو با موفقیت به دهکده «%s» پیوستی."
	MsgVillageChatNote      = "📝 پیام‌های شما در دهکده ارسال می‌شود. برای خروج /cancel بزنید یا از دکمه‌های منو استفاده کنید."
)

// Button labels
const (
	BtnVillageHub    = "🏘 دهکده من"
	BtnPlayGame      = "🎮 بازی کن!"
	BtnChatNow       = "💬 چت کن !"
	BtnLeaderboard   = "🏆 برترین ها"
	BtnFriends       = "👥 دوستان من"
	BtnProfile       = "👤 پروفایل من"
	BtnHelp          = "❓ راهنما"
	BtnReferral      = "📣 دعوت از دوستان"
	BtnCoins         = "💰 سکه"
	BtnIncreaseCoins = "➕ افزایش سکه"
	BtnShowBalance   = "📊 موجودی"

	BtnQuickMatch      = "🎲 بازی شانسی (Quick Match)"
	BtnQuiz            = "🧠 کوئیز اف کینگ"
	BtnTruthDare       = "🔥 جرعت و حقیقت"
	BtnOneVsOneRandom  = "👤 تک به تک (Random)"
	BtnPlayWithFriends = "⚔️ بازی با دوستان"
	BtnBetting         = "💰 شرط‌بندی (انتخاب مبلغ)"

	BtnCoinShop     = "🛍 فروشگاه سکه"
	BtnDailyBonus   = "🎁 جایزه روزانه (Daily Bonus)"
	BtnEditProfile  = "✏️ ویرایش پروفایل"
	BtnLikes        = "❤️ لایک‌ها"
	BtnEditLocation = "📍 ویرایش لوکیشن"
	BtnBlocks       = "🚫 بلاک شده‌ها"
	BtnSettings     = "⚙️ تنظیمات"
	BtnGameHistory  = "📜 تاریخچه بازی‌ها"

	BtnTodayTop   = "📅 برترینهای امروز"
	BtnWeekTop    = "🗓 برترینهای هفته"
	BtnAllTimeTop = "♾️ کل دوران"
	BtnQuizKings  = "🧠 سلاطین کوییز"
	BtnBraveOnes  = "🔥 شجاعترینها"
	BtnMyLeague   = "🏆 لیگ من"

	BtnInviteLink     = "🔗 لینک دعوت اختصاصی من"
	BtnFriendList     = "👥 لیست دوستان من"
	BtnFriendRequests = "🤝 درخواستهای دوستی"

	BtnNotifications = "🔔 اعلانها"
	BtnTutorials     = "📚 آموزش بازیها"
	BtnSupport       = "💬 پشتیبانی"
	BtnRules         = "⚖️ قوانین و مقررات"

	BtnCancel         = "❌ لغو"
	BtnEndChat        = "🔚 پایان چت"
	BtnSkip           = "⏭️ رد شو"
	BtnMale           = "👨 پسر"
	BtnFemale         = "👩 دختر"
	BtnAny            = "🌐 هر کدوم"
	BtnAccept         = "✅ قبول"
	BtnReject         = "❌ رد"
	BtnBack           = "🔙 بازگشت"
	BtnAdminPanel     = "🔧 پنل مدیریت"
	BtnAddQuestion    = "➕ افزودن سوال"
	BtnViewQuestions  = "📋 مشاهده سوالات"
	BtnUserManagement = "👥 مدیریت کاربران"

	// Truth or Dare New Buttons
	BtnTodAnonymous = "🕹 بازی با ناشناس"
	BtnTodFriends   = "👥 بازی با دوستان"
	BtnTodRegister  = "✍️ ثبت سوال"
	BtnTodHelp      = "📜 راهنما"
	BtnTodGirl      = "👧 بازی با دختر (۱۰ سکه)"
	BtnTodBoy       = "👦 بازی با پسر (۱۰ سکه)"
	BtnTodRandom    = "🎲 تصادفی (رایگان)"
	BtnTodAdvanced  = "⚙️ پیشرفته (۱۵ سکه)"
	BtnTodTruth     = "🤐 حقیقت"
	BtnTodTruth18   = "😈 حقیقت +۱۸"
	BtnTodDare      = "💪 جرات"
	BtnTodDare18    = "🔥 جرات +۱۸"
	BtnTodDone      = "✅ پاسخ داد / انجام شد"
	BtnTodSwap      = "🔄 تغییر سوال (۵ سکه)"
	BtnTodQuit      = "🚩 انصراف"

	BtnCreateRoom     = "🏛 ساخت روم"
	BtnSearchRoom     = "🔍 جستجوی روم"
	BtnRandomMatch    = "🎲 جستجوی تصادفی"
	BtnUserList       = "📋 لیست کاربران"
	BtnFilterRecent   = "🕒 چت های اخیر من"
	BtnFilterProvince = "📍 هم استانی ها"
	BtnFilterAge      = "🎂 هم سنی ها"
	BtnFilterNew      = "👶 کاربران جدید"
	BtnFilterNoChat   = "😶 بدون چت ها"
	BtnFilterAdvanced = "⚙️ جستجوی پیشرفته"
	BtnFilterNearMe   = "📍 نزدیک من"
	BtnBuyCoins       = "💎 خرید سکه"
	BtnIHavePaid      = "✅ واریز کردم"

	BtnCreateVillage      = "🆕 ساخت دهکده"
	BtnVillageLeaderboard = "🏅 برترین دهکده‌ها"
	BtnLeaveVillage       = "🚶 خروج از دهکده"
	BtnInviteToVillage    = "➕ دعوت به دهکده"
	BtnVillageChat        = "💬 چت دهکده"
	BtnVillageGame        = "🎮 بازی دهکده"
	BtnVillageTreasury    = "💰 خزانه دهکده"
	BtnVillageBuffs       = "✨ باف‌های دهکده"
	BtnVillageWar         = "⚔️ جنگ دهکده"
	BtnDonate             = "💎 اهدای سکه"
	BtnUpgradeXP          = "✨ ارتقا XP (5%)"
	BtnUpgradeCoin        = "💰 ارتقا سکه (5%)"
	BtnUpgradeShield      = "🛡 ارتقا سپر"
)

const (
	MsgVillageTreasury = "💰 **خزانه دهکده %s**\n\n💵 موجودی فعلی: %d سکه\n\nبرای ارتقای باف‌ها و شروع جنگ، دهکده نیاز به شارژ خزانه دارد.\n\n👇 می‌توانید به خزانه دهکده کمک کنید:"
	MsgVillageBuffs    = "✨ **باف‌های فعال دهکده**\n\n📈 افزایش XP: سطح %d (%d%%)\n💰 افزایش سکه: سطح %d (%d%%)\n🛡 سپر دفاعی: سطح %d\n\n💵 موجودی خزانه: %d\n\n👇 ارتقای باف‌ها (هزینه هر ارتقا: %d سکه):"
	MsgVillageWarInfo  = "⚔️ **وضعیت جنگ دهکده**\n\n%s\n\n👇 عملیات جنگ:"
)
