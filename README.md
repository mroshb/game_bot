# Telegram Game Bot - ربات تلگرام چت ناشناس و بازی

یک ربات تلگرام حرفه‌ای با قابلیت‌های چت ناشناس، matchmaking، سیستم سکه، و بازی‌های گروهی.

## ویژگی‌ها

### فاز 1 - چت ناشناس (پیاده‌سازی شده)
- ✅ ثبت نام کاربران با اطلاعات کامل
- ✅ سیستم matchmaking با الگوریتم هوشمند
- ✅ چت real-time بین کاربران
- ✅ سیستم سکه با تراکنش‌های امن
- ✅ مدیریت دوستان
- ✅ پنل مدیریت برای super admin
- ✅ امنیت FBI-level

### فاز 2 - بازی‌ها (در دست توسعه)
- ⏳ بازی حقیقت/جرات
- ⏳ کوییز چند نفره
- ⏳ اتاق‌های عمومی و خصوصی
- ⏳ سیستم امتیازدهی

## پیش‌نیازها

- Go 1.21 یا بالاتر
- PostgreSQL 13 یا بالاتر
- یک Bot Token از [@BotFather](https://t.me/BotFather)

## نصب و راه‌اندازی

### 1. کلون کردن پروژه

```bash
git clone https://github.com/mroshb/game_bot.git
cd game_bot
```

### 2. نصب dependencies

```bash
make deps
```

یا:

```bash
go mod download
go mod tidy
```

### 3. تنظیم environment variables

```bash
make env
```

سپس فایل `.env` را ویرایش کنید:

```env
# Telegram Bot
BOT_TOKEN=your_bot_token_here

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=gamebot
DB_PASSWORD=your_secure_password
DB_NAME=gamebot_db
DB_SSLMODE=disable  # 'require' in production

# Security
JWT_SECRET_KEY=your_jwt_secret_minimum_32_chars_here
AES_ENCRYPTION_KEY=your_aes_key_must_be_32_bytes!!

# Super Admin
SUPER_ADMIN_TELEGRAM_ID=your_telegram_id_here

# Application
APP_ENV=development
LOG_LEVEL=info
```

### 4. ایجاد دیتابیس

```bash
createdb gamebot_db
```

یا در PostgreSQL:

```sql
CREATE DATABASE gamebot_db;
```

### 5. اجرای برنامه

```bash
make dev
```

یا برای production:

```bash
make build
make run
```

## ساختار پروژه

```
game_bot/
├── cmd/bot/              # Entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database connection & migrations
│   ├── models/          # GORM models
│   ├── repositories/    # Database operations
│   ├── services/        # Business logic (آینده)
│   ├── handlers/        # Telegram bot handlers
│   ├── middleware/      # Auth, rate limiting (آینده)
│   ├── security/        # Encryption, JWT, sanitization
│   └── validators/      # Input validation (آینده)
├── pkg/
│   ├── logger/          # Structured logging
│   ├── errors/          # Custom error types
│   └── utils/           # Helper functions (آینده)
├── telegram/
│   ├── bot.go           # Bot initialization
│   ├── keyboards.go     # Inline keyboards
│   └── messages.go      # Persian message templates
└── scripts/             # Deployment scripts (آینده)
```

## دستورات Makefile

```bash
make build    # Build the application
make run      # Build and run
make dev      # Run in development mode
make test     # Run tests
make clean    # Clean build artifacts
make deps     # Install dependencies
make fmt      # Format code
make env      # Create .env from example
```

## امنیت

این پروژه با استانداردهای امنیتی FBI/MI6 طراحی شده:

- 🔐 AES-256 encryption برای data at rest
- 🔑 JWT authentication با HS-256
- 🛡️ Input sanitization و XSS protection
- 🔒 SQL injection prevention (GORM prepared statements)
- 🚫 No raw IP/phone storage (SHA-256 hashed)
- 📝 Audit logging برای sensitive operations

## مدیریت دیتابیس

پروژه از GORM auto-migration استفاده می‌کند. در اولین اجرا، تمام جداول به صورت خودکار ایجاد می‌شوند.

### Seed Data

5 سوال تستی به صورت خودکار در دیتابیس seed می‌شوند. برای مدیریت سوالات:

```sql
-- مشاهده سوالات
SELECT * FROM questions;

-- اضافه کردن سوال جدید
INSERT INTO questions (question_text, question_type, category, difficulty, points)
VALUES ('سوال شما', 'truth', 'دسته‌بندی', 'easy', 10);

-- حذف سوال
DELETE FROM questions WHERE id = 1;
```

## توسعه

### اضافه کردن فیچر جدید

1. Model را در `internal/models/` ایجاد کنید
2. Repository را در `internal/repositories/` بنویسید
3. Handler را در `internal/handlers/` پیاده‌سازی کنید
4. به `telegram/bot.go` اضافه کنید

### تست

```bash
go test ./...
```

## Deploy در Production

### 1. تنظیمات امنیتی

```env
APP_ENV=production
DB_SSLMODE=require
LOG_LEVEL=warn
```

### 2. Build

```bash
make build
```

### 3. اجرا با systemd

فایل `/etc/systemd/system/gamebot.service`:

```ini
[Unit]
Description=Telegram Game Bot
After=network.target postgresql.service

[Service]
Type=simple
User=gamebot
WorkingDirectory=/opt/gamebot
ExecStart=/opt/gamebot/bin/bot
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable gamebot
sudo systemctl start gamebot
sudo systemctl status gamebot
```

## مشکلات رایج

### خطای اتصال به دیتابیس

```bash
# بررسی وضعیت PostgreSQL
sudo systemctl status postgresql

# بررسی دسترسی
psql -U gamebot -d gamebot_db
```

### خطای Bot Token

مطمئن شوید Bot Token در `.env` صحیح است:

```bash
# تست با curl
curl https://api.telegram.org/bot<YOUR_TOKEN>/getMe
```

## مشارکت

این پروژه در حال توسعه است. برای مشارکت:

1. Fork کنید
2. Branch جدید بسازید
3. تغییرات را commit کنید
4. Pull Request بفرستید

## لایسنس

MIT License

## تماس

برای سوالات و پشتیبانی، با ما تماس بگیرید.

---

**نکته**: این پروژه در فاز اول توسعه است. بازی‌ها و فیچرهای اضافی در فاز بعدی اضافه خواهند شد.
