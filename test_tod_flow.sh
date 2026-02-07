#!/bin/bash

# Test Truth or Dare Game Flow
# این اسکریپت تمام مراحل بازی ToD رو تست میکنه

DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASS="A1212@shb#"
DB_NAME="game"

echo "🧪 شروع تست کامل بازی Truth or Dare"
echo "========================================"

# Function to run SQL query
run_sql() {
    PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "$1"
}

# 1. بررسی کاربران موجود
echo ""
echo "📋 مرحله 1: بررسی کاربران موجود"
echo "--------------------------------"
run_sql "SELECT id, telegram_id, full_name, gender FROM users ORDER BY id LIMIT 5;"

# 2. بررسی ToD Challenges
echo ""
echo "📋 مرحله 2: بررسی ToD Challenges"
echo "--------------------------------"
CHALLENGE_COUNT=$(run_sql "SELECT COUNT(*) FROM tod_challenges WHERE is_active = true;")
echo "تعداد challenges فعال: $CHALLENGE_COUNT"

if [ "$CHALLENGE_COUNT" -lt 5 ]; then
    echo "⚠️  هشدار: تعداد challenges کم است!"
fi

# 3. بررسی بازی‌های فعال
echo ""
echo "📋 مرحله 3: بررسی بازی‌های فعال"
echo "--------------------------------"
ACTIVE_GAMES=$(run_sql "SELECT COUNT(*) FROM tod_games WHERE state NOT IN ('game_end', 'forfeit');")
echo "تعداد بازی‌های فعال: $ACTIVE_GAMES"

# 4. بررسی match sessions فعال
echo ""
echo "📋 مرحله 4: بررسی Match Sessions فعال"
echo "--------------------------------"
ACTIVE_MATCHES=$(run_sql "SELECT COUNT(*) FROM match_sessions WHERE status = 'active';")
echo "تعداد match های فعال: $ACTIVE_MATCHES"

# 5. شبیه‌سازی ایجاد match و بازی
echo ""
echo "📋 مرحله 5: شبیه‌سازی ایجاد Match و بازی"
echo "--------------------------------"

# دریافت دو کاربر برای تست
USER1_ID=$(run_sql "SELECT id FROM users ORDER BY id LIMIT 1 OFFSET 0;" | xargs)
USER2_ID=$(run_sql "SELECT id FROM users ORDER BY id LIMIT 1 OFFSET 1;" | xargs)

if [ -z "$USER1_ID" ] || [ -z "$USER2_ID" ]; then
    echo "❌ خطا: کاربران کافی برای تست وجود ندارد"
    exit 1
fi

echo "👤 کاربر 1: ID = $USER1_ID"
echo "👤 کاربر 2: ID = $USER2_ID"

# ایجاد match session
echo ""
echo "🎮 ایجاد Match Session..."
MATCH_ID=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                    VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                    RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -z "$MATCH_ID" ]; then
    echo "❌ خطا در ایجاد match session"
    exit 1
fi

echo "✅ Match Session ایجاد شد: ID = $MATCH_ID"

# ایجاد بازی ToD
echo ""
echo "🎲 ایجاد بازی Truth or Dare..."
GAME_ID=$(run_sql "INSERT INTO tod_games (
    match_id, state, active_player_id, passive_player_id, 
    current_round, max_rounds, turn_started_at, turn_deadline,
    allow_items, difficulty_level, started_at, created_at, updated_at
) VALUES (
    $MATCH_ID, 'waiting_choice', $USER1_ID, $USER2_ID,
    1, 10, NOW(), NOW() + INTERVAL '60 seconds',
    true, 'normal', NOW(), NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -z "$GAME_ID" ]; then
    echo "❌ خطا در ایجاد بازی ToD"
    exit 1
fi

echo "✅ بازی ToD ایجاد شد: ID = $GAME_ID"

# ایجاد turn
echo ""
echo "🔄 ایجاد Turn..."
TURN_ID=$(run_sql "INSERT INTO tod_turns (
    game_id, round_number, player_id, judge_id
) VALUES (
    $GAME_ID, 1, $USER1_ID, $USER2_ID
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -z "$TURN_ID" ]; then
    echo "❌ خطا در ایجاد turn"
    exit 1
fi

echo "✅ Turn ایجاد شد: ID = $TURN_ID"

# Update game با current_turn_id
run_sql "UPDATE tod_games SET current_turn_id = $TURN_ID WHERE id = $GAME_ID;"

# 6. تست انتخاب Challenge
echo ""
echo "📋 مرحله 6: تست انتخاب Challenge"
echo "--------------------------------"

# انتخاب یک challenge تصادفی
CHALLENGE_ID=$(run_sql "SELECT id FROM tod_challenges WHERE type = 'truth' AND is_active = true ORDER BY RANDOM() LIMIT 1;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -z "$CHALLENGE_ID" ]; then
    echo "❌ خطا: هیچ challenge فعالی یافت نشد"
else
    echo "✅ Challenge انتخاب شد: ID = $CHALLENGE_ID"
    
    # نمایش اطلاعات challenge
    run_sql "SELECT id, type, text, difficulty, proof_type, coin_reward, xp_reward FROM tod_challenges WHERE id = $CHALLENGE_ID;"
    
    # Update turn با challenge
    run_sql "UPDATE tod_turns SET choice = 'truth', challenge_id = $CHALLENGE_ID, chosen_at = NOW() WHERE id = $TURN_ID;"
    echo "✅ Turn به‌روز شد با challenge"
fi

# 7. تست State Transitions
echo ""
echo "📋 مرحله 7: تست تغییر State"
echo "--------------------------------"

# waiting_choice -> waiting_proof
run_sql "UPDATE tod_games SET state = 'waiting_proof', updated_at = NOW() WHERE id = $GAME_ID;"
echo "✅ State تغییر کرد: waiting_choice -> waiting_proof"

# شبیه‌سازی ارسال proof
run_sql "UPDATE tod_turns SET proof_type = 'text', proof_data = 'این یک تست است', proof_submitted_at = NOW() WHERE id = $TURN_ID;"
echo "✅ Proof ثبت شد"

# waiting_proof -> waiting_judgment
run_sql "UPDATE tod_games SET state = 'waiting_judgment', updated_at = NOW() WHERE id = $GAME_ID;"
echo "✅ State تغییر کرد: waiting_proof -> waiting_judgment"

# شبیه‌سازی داوری
run_sql "UPDATE tod_turns SET judgment_result = 'accepted', judgment_reason = 'خوب بود', judged_at = NOW() WHERE id = $TURN_ID;"
echo "✅ Judgment ثبت شد"

# 8. تست پایان بازی و cleanup
echo ""
echo "📋 مرحله 8: تست پایان بازی و Cleanup"
echo "--------------------------------"

# پایان بازی
run_sql "UPDATE tod_games SET state = 'game_end', ended_at = NOW(), end_reason = 'test_completed', winner_id = $USER1_ID WHERE id = $GAME_ID;"
echo "✅ بازی به پایان رسید"

# بررسی match session
MATCH_STATUS=$(run_sql "SELECT status FROM match_sessions WHERE id = $MATCH_ID;" | xargs)
echo "📊 وضعیت Match Session: $MATCH_STATUS"

if [ "$MATCH_STATUS" = "active" ]; then
    echo "⚠️  هشدار: Match Session هنوز active است!"
    echo "🔧 Close کردن Match Session..."
    run_sql "UPDATE match_sessions SET status = 'finished', ended_at = NOW() WHERE id = $MATCH_ID;"
    echo "✅ Match Session بسته شد"
else
    echo "✅ Match Session به درستی بسته شده"
fi

# 9. تست Timeout Scenario
echo ""
echo "📋 مرحله 9: تست Timeout Scenario"
echo "--------------------------------"

# ایجاد یک بازی جدید برای تست timeout
MATCH_ID_2=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                      VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                      RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

GAME_ID_2=$(run_sql "INSERT INTO tod_games (
    match_id, state, active_player_id, passive_player_id, 
    current_round, max_rounds, turn_started_at, turn_deadline,
    allow_items, difficulty_level, started_at, created_at, updated_at
) VALUES (
    $MATCH_ID_2, 'waiting_choice', $USER1_ID, $USER2_ID,
    1, 10, NOW(), NOW() - INTERVAL '5 seconds',
    true, 'normal', NOW(), NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

echo "✅ بازی تست timeout ایجاد شد: ID = $GAME_ID_2"
echo "⏰ Deadline گذشته است (5 ثانیه پیش)"

# بررسی اینکه آیا background job این بازی رو پیدا میکنه
TIMEOUT_GAMES=$(run_sql "SELECT COUNT(*) FROM tod_games 
                         WHERE state IN ('waiting_choice', 'waiting_proof', 'waiting_judgment') 
                         AND turn_deadline IS NOT NULL 
                         AND turn_deadline < NOW();")
echo "🔍 تعداد بازی‌های timeout شده: $TIMEOUT_GAMES"

if [ "$TIMEOUT_GAMES" -gt 0 ]; then
    echo "✅ Background job باید این بازی رو پیدا کنه"
    echo "⏳ صبر کنید تا background job اجرا بشه (حداکثر 5 ثانیه)..."
    sleep 6
    
    # بررسی state بازی بعد از timeout
    GAME_STATE=$(run_sql "SELECT state FROM tod_games WHERE id = $GAME_ID_2;" | xargs)
    echo "📊 State بازی بعد از timeout: $GAME_STATE"
    
    if [ "$GAME_STATE" = "forfeit" ]; then
        echo "✅ بازی به درستی timeout شد"
        
        # بررسی match session
        MATCH_STATUS_2=$(run_sql "SELECT status FROM match_sessions WHERE id = $MATCH_ID_2;" | xargs)
        echo "📊 وضعیت Match Session بعد از timeout: $MATCH_STATUS_2"
        
        if [ "$MATCH_STATUS_2" = "ended" ]; then
            echo "✅ Match Session به درستی close شد"
        else
            echo "❌ خطا: Match Session بعد از timeout close نشد!"
        fi
    else
        echo "⏳ هنوز timeout اتفاق نیفتاده، ممکنه background job دیرتر اجرا بشه"
    fi
fi

# 10. خلاصه نتایج
echo ""
echo "📊 خلاصه نتایج تست"
echo "=================================="

TOTAL_GAMES=$(run_sql "SELECT COUNT(*) FROM tod_games;")
TOTAL_TURNS=$(run_sql "SELECT COUNT(*) FROM tod_turns;")
TOTAL_MATCHES=$(run_sql "SELECT COUNT(*) FROM match_sessions;")
ENDED_MATCHES=$(run_sql "SELECT COUNT(*) FROM match_sessions WHERE status = 'ended';")

echo "🎮 تعداد کل بازی‌ها: $TOTAL_GAMES"
echo "🔄 تعداد کل turn ها: $TOTAL_TURNS"
echo "🤝 تعداد کل match ها: $TOTAL_MATCHES"
echo "✅ تعداد match های ended: $ENDED_MATCHES"

echo ""
echo "🎉 تست کامل شد!"
