# 🚀 راهنمای سریع شروع کار

## گام 1: نصب PostgreSQL

### macOS
```bash
brew install postgresql@15
brew services start postgresql@15
```

### Ubuntu/Debian
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

## گام 2: ایجاد Database

```bash
# ورود به PostgreSQL
psql postgres

# ایجاد database و user
CREATE DATABASE gamebot_db;
CREATE USER gamebot WITH PASSWORD 'your_password_here';
GRANT ALL PRIVILEGES ON DATABASE gamebot_db TO gamebot;

# خروج
\q
```

## گام 3: دریافت Bot Token

1. به [@BotFather](https://t.me/BotFather) در تلگرام بروید
2. دستور `/newbot` را بفرستید
3. نام و username ربات را وارد کنید
4. Token را کپی کنید

## گام 4: تنظیم پروژه

```bash
# نصب dependencies
make deps

# ایجاد فایل .env
make env

# ویرایش .env
nano .env
```

### تنظیمات ضروری در `.env`:

```env
# Bot Token از BotFather
BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=gamebot
DB_PASSWORD=your_password_here
DB_NAME=gamebot_db
DB_SSLMODE=disable

# Security (حتماً تغییر دهید!)
JWT_SECRET_KEY=your_very_long_secret_key_at_least_32_characters_here
AES_ENCRYPTION_KEY=exactly_32_characters_key_here

# Super Admin (Telegram ID خودتان)
SUPER_ADMIN_TELEGRAM_ID=123456789

# Application
APP_ENV=development
LOG_LEVEL=debug
```

### 📝 یافتن Telegram ID خود

1. به [@userinfobot](https://t.me/userinfobot) بروید
2. `/start` را بفرستید
3. عدد `Id` را کپی کنید

## گام 5: اجرا

```bash
# اجرای development
make dev
```

اگر همه چیز درست باشد، باید این پیام را ببینید:

```
INFO  Bot started successfully  env=development
INFO  Authorized on account  username=YourBotName
```

## گام 6: تست ربات

1. ربات خود را در تلگرام پیدا کنید
2. `/start` را بفرستید
3. روی "📝 ثبت نام" کلیک کنید
4. مراحل ثبت نام را طی کنید

## ✅ Checklist

- [ ] PostgreSQL نصب و اجرا شده
- [ ] Database ایجاد شده
- [ ] Bot Token دریافت شده
- [ ] فایل `.env` تنظیم شده
- [ ] Telegram ID پیدا شده
- [ ] Dependencies نصب شده
- [ ] ربات اجرا شده
- [ ] ثبت نام تست شده

## 🐛 مشکلات رایج

### خطای اتصال به Database

```bash
# بررسی وضعیت PostgreSQL
brew services list  # macOS
sudo systemctl status postgresql  # Linux

# تست اتصال
psql -U gamebot -d gamebot_db
```

### خطای Bot Token

```
Error: 401 Unauthorized
```

**راه حل**: Bot Token را دوباره چک کنید.

### خطای Migration

```
Error: failed to run migrations
```

**راه حل**: مطمئن شوید database خالی است یا قبلاً migration نشده.

## 📚 دستورات مفید

```bash
# مشاهده logs
make dev

# Build برای production
make build

# اجرای production
make run

# پاک کردن build
make clean

# Format کردن کد
make fmt

# راهنما
make help
```

## 🎯 مرحله بعد

بعد از اجرای موفق:

1. ✅ ربات را تست کنید (ثبت نام، جستجو، چت)
2. ✅ با یک دوست تست matchmaking کنید
3. ✅ سیستم سکه را بررسی کنید
4. ✅ پنل ادمین را چک کنید (اگر super admin هستید)

## 💡 نکات

- در development mode، debug logging فعال است
- Auto-migration در اولین اجرا همه جداول را می‌سازد
- 5 سوال تستی به صورت خودکار seed می‌شود
- موجودی اولیه هر کاربر 100 سکه است

## 🆘 کمک

اگر مشکلی داشتید:

1. Logs را بررسی کنید
2. `.env` را دوباره چک کنید
3. Database connection را تست کنید
4. Bot Token را verify کنید

---

**موفق باشید!** 🚀
