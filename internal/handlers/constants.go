package handlers

// Button texts matching telegram/messages.go
// Ideally these should be in a shared package, but for now we mirror them here
// to avoid circular dependency with telegram package
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
	BtnInventory      = "🎒 کوله‌پشتی (Inventory)"
	BtnHistory        = "📜 تاریخچه بازی‌ها"

	BtnNotifications = "🔔 اعلانها"
	BtnTutorials     = "📚 آموزش بازیها"
	BtnSupport       = "💬 پشتیبانی"
	BtnRules         = "⚖️ قوانین و مقررات"

	BtnRegister       = "📝 ثبت نام"
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
	BtnBuyCoins       = "💎 خرید سکه"
	BtnIHavePaid      = "✅ واریز کردم"

	BtnCreateVillage      = "🆕 ساخت دهکده"
	BtnVillageLeaderboard = "🏅 برترین دهکده‌ها"
	BtnLeaveVillage       = "🚶 خروج از دهکده"
	BtnInviteToVillage    = "➕ دعوت به دهکده"
	BtnVillageChat        = "💬 چت دهکده"
	BtnVillageGame        = "🎮 بازی دهکده"

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

	BtnVillageTreasury = "💰 خزانه دهکده"
	BtnVillageBuffs    = "✨ باف‌های دهکده"
	BtnVillageWar      = "⚔️ جنگ دهکده"

	MsgVillageTreasury = "💰 **خزانه دهکده %s**\n\n💵 موجودی فعلی: %d سکه\n\nبرای ارتقای باف‌ها و شروع جنگ، دهکده نیاز به شارژ خزانه دارد.\n\n👇 می‌توانید به خزانه دهکده کمک کنید:"
	MsgVillageBuffs    = "✨ **باف‌های فعال دهکده**\n\n📈 افزایش XP: سطح %d (%d%%)\n💰 افزایش سکه: سطح %d (%d%%)\n🛡 سپر دفاعی: سطح %d\n\n💵 موجودی خزانه: %d\n\n👇 ارتقای باف‌ها (هزینه هر ارتقا: %d سکه):"
	MsgVillageWarInfo  = "⚔️ **وضعیت جنگ دهکده**\n\n%s\n\n👇 عملیات جنگ:"

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
	BtnEndGame      = "🔚 پایان بازی"

	MsgErrorGeneric      = "❌ متاسفانه خطایی رخ داده است. لطفاً دوباره تلاش کنید."
	MsgInsufficientCoins = "⚠️ سکه کافی نداری! موجودی فعلی: %d"
	MsgAlreadySearching  = "🔍 شما در حال حاضر در صف انتظار هستید."
	MsgAlreadyInMatch    = "⚠️ شما در حال حاضر در یک بازی فعال هستید."
)
