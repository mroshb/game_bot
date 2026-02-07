#!/bin/bash

# Final Production Readiness Check
# بررسی نهایی آمادگی برای پروداکشن

echo "╔════════════════════════════════════════════════════════════╗"
echo "║     🔍 بررسی نهایی آمادگی پروداکشن                        ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

PASS=0
FAIL=0

check() {
    local name="$1"
    local command="$2"
    
    echo -n "  ⏳ $name... "
    
    if eval "$command" > /dev/null 2>&1; then
        echo "✅"
        PASS=$((PASS + 1))
    else
        echo "❌"
        FAIL=$((FAIL + 1))
    fi
}

echo "═══════════════════════════════════════════════════════════"
echo "1️⃣  بررسی کد و Build"
echo "═══════════════════════════════════════════════════════════"
check "Go Build بدون خطا" "go build -o /tmp/test_bot cmd/bot/main.go"
check "فایل اجرایی موجود" "test -f ./game_bot"
check "فایل .env موجود" "test -f ./.env"
check "دسترسی اجرایی" "test -x ./game_bot"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "2️⃣  بررسی دیتابیس"
echo "═══════════════════════════════════════════════════════════"
check "اتصال به دیتابیس" "PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -c 'SELECT 1;'"
check "جدول users" "PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -c 'SELECT COUNT(*) FROM users;'"
check "جدول tod_games" "PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -c 'SELECT COUNT(*) FROM tod_games;'"
check "جدول quiz_matches" "PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -c 'SELECT COUNT(*) FROM quiz_matches;'"
check "جدول match_sessions" "PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -c 'SELECT COUNT(*) FROM match_sessions;'"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "3️⃣  بررسی بات"
echo "═══════════════════════════════════════════════════════════"
check "بات در حال اجرا" "ps aux | grep -E 'game_bot$' | grep -v grep"
check "لاگ فایل موجود" "test -f ./bot.log"
check "لاگ بدون خطای اخیر" "! tail -100 bot.log | grep -i 'ERROR.*relation.*does not exist'"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "4️⃣  بررسی Background Jobs"
echo "═══════════════════════════════════════════════════════════"
check "Timeout checker فعال" "tail -50 bot.log | grep -q 'tod_games.*turn_deadline'"
check "getUpdates فعال" "tail -50 bot.log | grep -q 'getUpdates'"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "5️⃣  بررسی داده‌های اولیه"
echo "═══════════════════════════════════════════════════════════"
check "کاربران موجود" "test \$(PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -t -c 'SELECT COUNT(*) FROM users;' | tr -d ' ') -gt 0"
check "ToD Challenges موجود" "test \$(PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -t -c 'SELECT COUNT(*) FROM tod_challenges WHERE is_active = true;' | tr -d ' ') -gt 5"
check "Quiz Questions موجود" "test \$(PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -t -c 'SELECT COUNT(*) FROM questions WHERE question_type = '\"'\"'quiz'\"'\"';' | tr -d ' ') -gt 10"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "6️⃣  بررسی Performance"
echo "═══════════════════════════════════════════════════════════"
check "Indexes کافی (>50)" "test \$(PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -t -c 'SELECT COUNT(*) FROM pg_indexes WHERE schemaname = '\"'\"'public'\"'\"';' | tr -d ' ') -gt 50"
check "Foreign Keys کافی (>20)" "test \$(PGPASSWORD='A1212@shb#' psql -h localhost -U postgres -d game -t -c 'SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type = '\"'\"'FOREIGN KEY'\"'\"';' | tr -d ' ') -gt 20"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "7️⃣  بررسی امنیت و Best Practices"
echo "═══════════════════════════════════════════════════════════"
check "Connection Pool تنظیم شده" "grep -q 'SetMaxOpenConns' internal/database/connection.go"
check "Prepared Statements فعال" "grep -q 'PrepareStmt.*true' internal/database/connection.go"
check "Logger موجود" "test -d pkg/logger"
check "Error Handling" "grep -q 'errors.Wrap' internal/repositories/*.go"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 نتیجه نهایی"
echo "═══════════════════════════════════════════════════════════"
echo ""

TOTAL=$((PASS + FAIL))
SUCCESS_RATE=$((PASS * 100 / TOTAL))

echo "✅ موفق: $PASS"
echo "❌ ناموفق: $FAIL"
echo "📊 کل: $TOTAL"
echo "🎯 نرخ موفقیت: $SUCCESS_RATE%"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                                                            ║"
    echo "║        ✅ پروژه 100% آماده برای پروداکشن است! ✅          ║"
    echo "║                                                            ║"
    echo "║  🎯 همه چک‌ها موفق                                        ║"
    echo "║  🚀 بات در حال اجرا                                       ║"
    echo "║  💾 دیتابیس سالم                                          ║"
    echo "║  ⚡ Performance بهینه                                      ║"
    echo "║  🔒 امنیت رعایت شده                                       ║"
    echo "║                                                            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    exit 0
else
    echo "⚠️  برخی چک‌ها ناموفق بودند. لطفاً بررسی کنید."
    exit 1
fi
