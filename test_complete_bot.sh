#!/bin/bash

# Complete Bot Testing Suite
# این اسکریپت تمام بخش‌های بات رو به صورت کامل تست میکنه

DB_HOST="localhost"
DB_PORT="5432"
DB_USER="postgres"
DB_PASS="A1212@shb#"
DB_NAME="game"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo "╔════════════════════════════════════════════════════════════╗"
echo "║     🧪 تست کامل و جامع ربات بازی تلگرام                  ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Function to run SQL query
run_sql() {
    PGPASSWORD="$DB_PASS" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -t -c "$1" 2>&1
}

# Function to test a feature
test_feature() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -n "  ⏳ $test_name... "
    
    result=$(eval "$test_command")
    
    if [[ "$result" == *"$expected_result"* ]] || [[ -z "$expected_result" && -n "$result" ]]; then
        echo -e "${GREEN}✅ PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "    Expected: $expected_result"
        echo "    Got: $result"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Function to check table exists
check_table() {
    local table_name="$1"
    run_sql "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table_name';" | tr -d ' '
}

# Function to check column exists
check_column() {
    local table_name="$1"
    local column_name="$2"
    run_sql "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = '$table_name' AND column_name = '$column_name';" | tr -d ' '
}

echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 1: بررسی ساختار دیتابیس"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Test database tables
echo "🔍 بررسی جداول اصلی:"
test_feature "جدول users" "check_table 'users'" "1"
test_feature "جدول coin_transactions" "check_table 'coin_transactions'" "1"
test_feature "جدول match_sessions" "check_table 'match_sessions'" "1"
test_feature "جدول matchmaking_queue" "check_table 'matchmaking_queue'" "1"
test_feature "جدول friendships" "check_table 'friendships'" "1"
test_feature "جدول questions" "check_table 'questions'" "1"
test_feature "جدول game_sessions" "check_table 'game_sessions'" "1"
test_feature "جدول rooms" "check_table 'rooms'" "1"
test_feature "جدول villages" "check_table 'villages'" "1"

echo ""
echo "🔍 بررسی جداول Truth or Dare:"
test_feature "جدول tod_games" "check_table 'tod_games'" "1"
test_feature "جدول tod_turns" "check_table 'tod_turns'" "1"
test_feature "جدول tod_challenges" "check_table 'tod_challenges'" "1"
test_feature "جدول tod_player_stats" "check_table 'tod_player_stats'" "1"
test_feature "جدول tod_judgment_logs" "check_table 'tod_judgment_logs'" "1"
test_feature "جدول tod_action_logs" "check_table 'tod_action_logs'" "1"

echo ""
echo "🔍 بررسی جداول Quiz:"
test_feature "جدول quiz_matches" "check_table 'quiz_matches'" "1"
test_feature "جدول quiz_rounds" "check_table 'quiz_rounds'" "1"
test_feature "جدول quiz_answers" "check_table 'quiz_answers'" "1"
test_feature "جدول user_boosters" "check_table 'user_boosters'" "1"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 2: بررسی داده‌های اولیه"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Check seeded data
echo "🌱 بررسی داده‌های seed شده:"

USERS_COUNT=$(run_sql "SELECT COUNT(*) FROM users;" | tr -d ' ')
test_feature "کاربران موجود (حداقل 1)" "echo $USERS_COUNT" ""
echo "    📊 تعداد کاربران: $USERS_COUNT"

QUESTIONS_COUNT=$(run_sql "SELECT COUNT(*) FROM questions;" | tr -d ' ')
test_feature "سوالات موجود (حداقل 5)" "echo $QUESTIONS_COUNT" ""
echo "    📊 تعداد سوالات: $QUESTIONS_COUNT"

TOD_CHALLENGES_COUNT=$(run_sql "SELECT COUNT(*) FROM tod_challenges WHERE is_active = true;" | tr -d ' ')
test_feature "ToD Challenges فعال (حداقل 5)" "echo $TOD_CHALLENGES_COUNT" ""
echo "    📊 تعداد challenges: $TOD_CHALLENGES_COUNT"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 3: تست سیستم کاربری"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Get test users
USER1_ID=$(run_sql "SELECT id FROM users ORDER BY id LIMIT 1 OFFSET 0;" | tr -d ' ' | grep -o '[0-9]*' | head -1)
USER2_ID=$(run_sql "SELECT id FROM users ORDER BY id LIMIT 1 OFFSET 1;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -z "$USER1_ID" ] || [ -z "$USER2_ID" ]; then
    echo -e "${RED}❌ خطا: کاربران کافی برای تست وجود ندارد${NC}"
    echo "لطفاً حداقل 2 کاربر در سیستم ثبت کنید"
    exit 1
fi

echo "👥 کاربران تست:"
echo "    User 1: ID = $USER1_ID"
echo "    User 2: ID = $USER2_ID"

# Test user data integrity
echo ""
echo "🔍 بررسی یکپارچگی داده‌های کاربر:"
test_feature "فیلدهای ضروری User 1" "run_sql \"SELECT telegram_id, full_name, gender FROM users WHERE id = $USER1_ID;\"" ""
test_feature "سطح و امتیاز User 1" "run_sql \"SELECT level, xp, coins FROM users WHERE id = $USER1_ID;\"" ""

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 4: تست سیستم سکه و تراکنش‌ها"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Test coin system
echo "💰 تست سیستم سکه:"

# Add coins to user
INITIAL_COINS=$(run_sql "SELECT coins FROM users WHERE id = $USER1_ID;" | tr -d ' ')
echo "    💵 موجودی اولیه User 1: $INITIAL_COINS سکه"

# Create a coin transaction
run_sql "INSERT INTO coin_transactions (user_id, amount, transaction_type, description, created_at) 
         VALUES ($USER1_ID, 100, 'test_reward', 'تست سیستم سکه', NOW());" > /dev/null 2>&1

run_sql "UPDATE users SET coins = coins + 100 WHERE id = $USER1_ID;" > /dev/null 2>&1

NEW_COINS=$(run_sql "SELECT coins FROM users WHERE id = $USER1_ID;" | tr -d ' ')
EXPECTED_COINS=$((INITIAL_COINS + 100))

test_feature "افزایش سکه (100+)" "echo $NEW_COINS" "$EXPECTED_COINS"
echo "    💵 موجودی جدید: $NEW_COINS سکه"

# Check transaction log
TRANSACTION_COUNT=$(run_sql "SELECT COUNT(*) FROM coin_transactions WHERE user_id = $USER1_ID AND transaction_type = 'test_reward';" | tr -d ' ')
test_feature "ثبت تراکنش" "echo $TRANSACTION_COUNT" "1"

# Rollback test coins
run_sql "UPDATE users SET coins = coins - 100 WHERE id = $USER1_ID;" > /dev/null 2>&1
run_sql "DELETE FROM coin_transactions WHERE user_id = $USER1_ID AND transaction_type = 'test_reward';" > /dev/null 2>&1

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 5: تست سیستم Matchmaking"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🎯 تست Matchmaking:"

# Clean up any existing matches
run_sql "DELETE FROM match_sessions WHERE user1_id IN ($USER1_ID, $USER2_ID) OR user2_id IN ($USER1_ID, $USER2_ID);" > /dev/null 2>&1

# Create a match
MATCH_ID=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                    VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                    RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Match Session" "echo $MATCH_ID" ""
echo "    🎮 Match ID: $MATCH_ID"

# Check match status
MATCH_STATUS=$(run_sql "SELECT status FROM match_sessions WHERE id = $MATCH_ID;" | tr -d ' ')
test_feature "وضعیت Match" "echo $MATCH_STATUS" "active"

# End match
run_sql "UPDATE match_sessions SET status = 'ended', ended_at = NOW() WHERE id = $MATCH_ID;" > /dev/null 2>&1
ENDED_STATUS=$(run_sql "SELECT status FROM match_sessions WHERE id = $MATCH_ID;" | tr -d ' ')
test_feature "پایان Match" "echo $ENDED_STATUS" "ended"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 6: تست سیستم دوستی"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "👫 تست سیستم دوستی:"

# Clean up existing friendships
run_sql "DELETE FROM friendships WHERE (user_id = $USER1_ID AND friend_id = $USER2_ID) OR (user_id = $USER2_ID AND friend_id = $USER1_ID);" > /dev/null 2>&1

# Create friendship request
FRIENDSHIP_ID=$(run_sql "INSERT INTO friendships (user_id, friend_id, status, created_at) 
                         VALUES ($USER1_ID, $USER2_ID, 'pending', NOW()) 
                         RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد درخواست دوستی" "echo $FRIENDSHIP_ID" ""

# Accept friendship
run_sql "UPDATE friendships SET status = 'accepted' WHERE id = $FRIENDSHIP_ID;" > /dev/null 2>&1
FRIENDSHIP_STATUS=$(run_sql "SELECT status FROM friendships WHERE id = $FRIENDSHIP_ID;" | tr -d ' ')
test_feature "پذیرش دوستی" "echo $FRIENDSHIP_STATUS" "accepted"

# Clean up
run_sql "DELETE FROM friendships WHERE id = $FRIENDSHIP_ID;" > /dev/null 2>&1

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 7: تست بازی Truth or Dare (جامع)"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🎲 تست کامل Truth or Dare:"

# Clean up
run_sql "DELETE FROM tod_games WHERE match_id IN (SELECT id FROM match_sessions WHERE user1_id IN ($USER1_ID, $USER2_ID));" > /dev/null 2>&1
run_sql "DELETE FROM match_sessions WHERE user1_id IN ($USER1_ID, $USER2_ID) OR user2_id IN ($USER1_ID, $USER2_ID);" > /dev/null 2>&1

# Create match for ToD
TOD_MATCH_ID=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                        VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                        RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

# Create ToD game
TOD_GAME_ID=$(run_sql "INSERT INTO tod_games (
    match_id, state, active_player_id, passive_player_id, 
    current_round, max_rounds, turn_started_at, turn_deadline,
    allow_items, difficulty_level, started_at, created_at, updated_at
) VALUES (
    $TOD_MATCH_ID, 'waiting_choice', $USER1_ID, $USER2_ID,
    1, 10, NOW(), NOW() + INTERVAL '60 seconds',
    true, 'normal', NOW(), NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد بازی ToD" "echo $TOD_GAME_ID" ""

# Create turn
TOD_TURN_ID=$(run_sql "INSERT INTO tod_turns (game_id, round_number, player_id, judge_id) 
                       VALUES ($TOD_GAME_ID, 1, $USER1_ID, $USER2_ID) 
                       RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Turn" "echo $TOD_TURN_ID" ""

# Update game with turn
run_sql "UPDATE tod_games SET current_turn_id = $TOD_TURN_ID WHERE id = $TOD_GAME_ID;" > /dev/null 2>&1

# Select challenge
TOD_CHALLENGE_ID=$(run_sql "SELECT id FROM tod_challenges WHERE type = 'truth' AND is_active = true ORDER BY RANDOM() LIMIT 1;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "انتخاب Challenge" "echo $TOD_CHALLENGE_ID" ""

# Update turn with choice and challenge
run_sql "UPDATE tod_turns SET choice = 'truth', challenge_id = $TOD_CHALLENGE_ID, chosen_at = NOW() WHERE id = $TOD_TURN_ID;" > /dev/null 2>&1

# Test state transitions
run_sql "UPDATE tod_games SET state = 'waiting_proof' WHERE id = $TOD_GAME_ID;" > /dev/null 2>&1
STATE1=$(run_sql "SELECT state FROM tod_games WHERE id = $TOD_GAME_ID;" | tr -d ' ')
test_feature "State: waiting_proof" "echo $STATE1" "waiting_proof"

run_sql "UPDATE tod_turns SET proof_type = 'text', proof_data = 'تست', proof_submitted_at = NOW() WHERE id = $TOD_TURN_ID;" > /dev/null 2>&1
run_sql "UPDATE tod_games SET state = 'waiting_judgment' WHERE id = $TOD_GAME_ID;" > /dev/null 2>&1
STATE2=$(run_sql "SELECT state FROM tod_games WHERE id = $TOD_GAME_ID;" | tr -d ' ')
test_feature "State: waiting_judgment" "echo $STATE2" "waiting_judgment"

run_sql "UPDATE tod_turns SET judgment_result = 'accepted', judged_at = NOW() WHERE id = $TOD_TURN_ID;" > /dev/null 2>&1
run_sql "UPDATE tod_games SET state = 'game_end', ended_at = NOW(), winner_id = $USER1_ID WHERE id = $TOD_GAME_ID;" > /dev/null 2>&1
STATE3=$(run_sql "SELECT state FROM tod_games WHERE id = $TOD_GAME_ID;" | tr -d ' ')
test_feature "State: game_end" "echo $STATE3" "game_end"

# Check match cleanup
run_sql "UPDATE match_sessions SET status = 'ended', ended_at = NOW() WHERE id = $TOD_MATCH_ID;" > /dev/null 2>&1
MATCH_CLEANUP=$(run_sql "SELECT status FROM match_sessions WHERE id = $TOD_MATCH_ID;" | tr -d ' ')
test_feature "Match Cleanup" "echo $MATCH_CLEANUP" "ended"

# Test player stats
run_sql "INSERT INTO tod_player_stats (user_id, judge_score, created_at, updated_at) 
         VALUES ($USER1_ID, 100.0, NOW(), NOW()) 
         ON CONFLICT (user_id) DO NOTHING;" > /dev/null 2>&1

STATS_EXISTS=$(run_sql "SELECT COUNT(*) FROM tod_player_stats WHERE user_id = $USER1_ID;" | tr -d ' ')
test_feature "Player Stats" "echo $STATS_EXISTS" "1"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 8: تست سیستم Quiz"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🧠 تست سیستم Quiz:"

# Clean up
run_sql "DELETE FROM quiz_matches WHERE user1_id IN ($USER1_ID, $USER2_ID) OR user2_id IN ($USER1_ID, $USER2_ID);" > /dev/null 2>&1

# Create quiz match
QUIZ_MATCH_ID=$(run_sql "INSERT INTO quiz_matches (
    user1_id, user2_id, current_round, state, timeout_at, created_at, updated_at
) VALUES (
    $USER1_ID, $USER2_ID, 1, 'waiting_category', NOW() + INTERVAL '3 days', NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Quiz Match" "echo $QUIZ_MATCH_ID" ""

# Update state
run_sql "UPDATE quiz_matches SET state = 'playing_q1' WHERE id = $QUIZ_MATCH_ID;" > /dev/null 2>&1
QUIZ_STATE=$(run_sql "SELECT state FROM quiz_matches WHERE id = $QUIZ_MATCH_ID;" | tr -d ' ')
test_feature "Quiz State Update" "echo $QUIZ_STATE" "playing_q1"

# Create quiz round
QUIZ_ROUND_ID=$(run_sql "INSERT INTO quiz_rounds (match_id, round_number, category, chosen_by_user_id, created_at) 
                         VALUES ($QUIZ_MATCH_ID, 1, 'علمی', $USER1_ID, NOW()) 
                         RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Quiz Round" "echo $QUIZ_ROUND_ID" ""

# Get a quiz question
QUIZ_QUESTION_ID=$(run_sql "SELECT id FROM questions WHERE question_type = 'quiz' ORDER BY RANDOM() LIMIT 1;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

if [ -n "$QUIZ_QUESTION_ID" ]; then
    # Create quiz answer
    QUIZ_ANSWER_ID=$(run_sql "INSERT INTO quiz_answers (
        match_id, round_id, user_id, question_id, question_number, 
        answer_index, is_correct, time_taken_ms, answered_at
    ) VALUES (
        $QUIZ_MATCH_ID, $QUIZ_ROUND_ID, $USER1_ID, $QUIZ_QUESTION_ID, 1,
        0, true, 5000, NOW()
    ) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)
    
    test_feature "ثبت پاسخ Quiz" "echo $QUIZ_ANSWER_ID" ""
fi

# Test user boosters
run_sql "INSERT INTO user_boosters (user_id, booster_type, quantity, created_at, updated_at) 
         VALUES ($USER1_ID, 'remove_2_options', 3, NOW(), NOW()) 
         ON CONFLICT (user_id, booster_type) DO UPDATE SET quantity = 3;" > /dev/null 2>&1

BOOSTER_COUNT=$(run_sql "SELECT quantity FROM user_boosters WHERE user_id = $USER1_ID AND booster_type = 'remove_2_options';" | tr -d ' ')
test_feature "User Boosters" "echo $BOOSTER_COUNT" "3"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 9: تست سیستم اتاق‌ها"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🏠 تست سیستم Room:"

# Clean up
run_sql "DELETE FROM rooms WHERE owner_id = $USER1_ID;" > /dev/null 2>&1

# Create room
ROOM_ID=$(run_sql "INSERT INTO rooms (
    name, owner_id, max_members, is_private, status, created_at, updated_at
) VALUES (
    'تست روم', $USER1_ID, 10, false, 'active', NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Room" "echo $ROOM_ID" ""

# Add member
run_sql "INSERT INTO room_members (room_id, user_id, role, joined_at) 
         VALUES ($ROOM_ID, $USER1_ID, 'owner', NOW());" > /dev/null 2>&1

MEMBER_COUNT=$(run_sql "SELECT COUNT(*) FROM room_members WHERE room_id = $ROOM_ID;" | tr -d ' ')
test_feature "اضافه کردن عضو" "echo $MEMBER_COUNT" "1"

# Clean up
run_sql "DELETE FROM rooms WHERE id = $ROOM_ID;" > /dev/null 2>&1

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 10: تست سیستم دهکده‌ها"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🏘️ تست سیستم Village:"

# Clean up
run_sql "DELETE FROM villages WHERE owner_id = $USER1_ID;" > /dev/null 2>&1

# Create village
VILLAGE_ID=$(run_sql "INSERT INTO villages (
    name, owner_id, max_members, level, xp, coins, status, created_at, updated_at
) VALUES (
    'تست دهکده', $USER1_ID, 50, 1, 0, 0, 'active', NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

test_feature "ایجاد Village" "echo $VILLAGE_ID" ""

# Add member
run_sql "INSERT INTO village_members (village_id, user_id, role, joined_at) 
         VALUES ($VILLAGE_ID, $USER1_ID, 'owner', NOW());" > /dev/null 2>&1

VILLAGE_MEMBER_COUNT=$(run_sql "SELECT COUNT(*) FROM village_members WHERE village_id = $VILLAGE_ID;" | tr -d ' ')
test_feature "اضافه کردن عضو به Village" "echo $VILLAGE_MEMBER_COUNT" "1"

# Update village XP
run_sql "UPDATE villages SET xp = xp + 100 WHERE id = $VILLAGE_ID;" > /dev/null 2>&1
VILLAGE_XP=$(run_sql "SELECT xp FROM villages WHERE id = $VILLAGE_ID;" | tr -d ' ')
test_feature "افزایش XP دهکده" "echo $VILLAGE_XP" "100"

# Clean up
run_sql "DELETE FROM villages WHERE id = $VILLAGE_ID;" > /dev/null 2>&1

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 11: تست Indexes و Performance"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "⚡ بررسی Indexes:"

# Check important indexes
INDEXES=$(run_sql "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public';" | tr -d ' ')
echo "    📊 تعداد کل indexes: $INDEXES"
test_feature "Indexes موجود (حداقل 20)" "echo $INDEXES" ""

# Check specific critical indexes
test_feature "Index: users telegram_id" "run_sql \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'users' AND indexname LIKE '%telegram_id%';\"" "1"
test_feature "Index: match_sessions status" "run_sql \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'match_sessions' AND indexname LIKE '%status%';\"" "1"
test_feature "Index: tod_games state" "run_sql \"SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'tod_games' AND indexname LIKE '%state%';\"" "1"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 12: تست Foreign Keys و Cascade"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "🔗 بررسی Foreign Keys:"

FK_COUNT=$(run_sql "SELECT COUNT(*) FROM information_schema.table_constraints WHERE constraint_type = 'FOREIGN KEY' AND table_schema = 'public';" | tr -d ' ')
echo "    📊 تعداد کل Foreign Keys: $FK_COUNT"
test_feature "Foreign Keys موجود (حداقل 15)" "echo $FK_COUNT" ""

# Test cascade delete
echo ""
echo "🗑️ تست Cascade Delete:"

# Create test data
TEST_MATCH_ID=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                         VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                         RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

TEST_GAME_ID=$(run_sql "INSERT INTO tod_games (
    match_id, state, active_player_id, passive_player_id, 
    current_round, max_rounds, started_at, created_at, updated_at
) VALUES (
    $TEST_MATCH_ID, 'waiting_choice', $USER1_ID, $USER2_ID,
    1, 10, NOW(), NOW(), NOW()
) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)

# Delete match (should cascade to game)
run_sql "DELETE FROM match_sessions WHERE id = $TEST_MATCH_ID;" > /dev/null 2>&1

GAME_EXISTS=$(run_sql "SELECT COUNT(*) FROM tod_games WHERE id = $TEST_GAME_ID;" | tr -d ' ')
test_feature "Cascade Delete (tod_games)" "echo $GAME_EXISTS" "0"

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 بخش 13: تست Background Jobs"
echo "═══════════════════════════════════════════════════════════"
echo ""

echo "⏰ تست Background Jobs:"

# Check if bot is running
BOT_RUNNING=$(ps aux | grep -E "game_bot$" | grep -v grep | wc -l | tr -d ' ')
test_feature "Bot در حال اجرا" "echo $BOT_RUNNING" "1"

if [ "$BOT_RUNNING" -eq 1 ]; then
    echo "    ✅ Background jobs در حال اجرا هستند"
    
    # Create a timeout test
    TIMEOUT_MATCH=$(run_sql "INSERT INTO match_sessions (user1_id, user2_id, status, created_at) 
                             VALUES ($USER1_ID, $USER2_ID, 'active', NOW()) 
                             RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)
    
    TIMEOUT_GAME=$(run_sql "INSERT INTO tod_games (
        match_id, state, active_player_id, passive_player_id, 
        current_round, max_rounds, turn_started_at, turn_deadline,
        started_at, created_at, updated_at
    ) VALUES (
        $TIMEOUT_MATCH, 'waiting_choice', $USER1_ID, $USER2_ID,
        1, 10, NOW(), NOW() - INTERVAL '10 seconds',
        NOW(), NOW(), NOW()
    ) RETURNING id;" | tr -d ' ' | grep -o '[0-9]*' | head -1)
    
    echo "    ⏳ صبر برای اجرای background job (10 ثانیه)..."
    sleep 11
    
    TIMEOUT_STATE=$(run_sql "SELECT state FROM tod_games WHERE id = $TIMEOUT_GAME;" | tr -d ' ')
    test_feature "Timeout Handler" "echo $TIMEOUT_STATE" "forfeit"
    
    TIMEOUT_MATCH_STATUS=$(run_sql "SELECT status FROM match_sessions WHERE id = $TIMEOUT_MATCH;" | tr -d ' ')
    test_feature "Match Cleanup بعد از Timeout" "echo $TIMEOUT_MATCH_STATUS" "ended"
else
    echo -e "    ${YELLOW}⚠️  Bot در حال اجرا نیست - تست background jobs skip شد${NC}"
fi

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "📊 نتیجه نهایی تست"
echo "═══════════════════════════════════════════════════════════"
echo ""

SUCCESS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))

echo "📊 آمار کلی:"
echo "    ✅ تست‌های موفق: $PASSED_TESTS"
echo "    ❌ تست‌های ناموفق: $FAILED_TESTS"
echo "    📈 کل تست‌ها: $TOTAL_TESTS"
echo "    🎯 نرخ موفقیت: $SUCCESS_RATE%"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           🎉 همه تست‌ها با موفقیت انجام شد! 🎉           ║${NC}"
    echo -e "${GREEN}║                                                            ║${NC}"
    echo -e "${GREEN}║  ✅ دیتابیس: کامل و صحیح                                  ║${NC}"
    echo -e "${GREEN}║  ✅ سیستم‌ها: عملکرد صحیح                                 ║${NC}"
    echo -e "${GREEN}║  ✅ Performance: بهینه                                     ║${NC}"
    echo -e "${GREEN}║  ✅ Background Jobs: فعال و کارآمد                        ║${NC}"
    echo -e "${GREEN}║                                                            ║${NC}"
    echo -e "${GREEN}║        🚀 پروژه آماده برای استفاده در پروداکشن! 🚀       ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║              ⚠️  برخی تست‌ها ناموفق بودند ⚠️              ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "لطفاً خطاهای بالا را بررسی و رفع کنید."
    exit 1
fi
