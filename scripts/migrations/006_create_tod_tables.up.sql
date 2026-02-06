-- Migration: Create Truth or Dare tables
-- File: scripts/migrations/006_create_tod_tables.up.sql

-- Challenges table
CREATE TABLE IF NOT EXISTS tod_challenges (
    id SERIAL PRIMARY KEY,
    type VARCHAR(10) NOT NULL,
    text TEXT NOT NULL,
    difficulty VARCHAR(20),
    category VARCHAR(50),
    gender_target VARCHAR(10),
    relation_level VARCHAR(20),
    proof_type VARCHAR(20) NOT NULL,
    proof_hint TEXT,
    xp_reward INT DEFAULT 20,
    coin_reward INT DEFAULT 10,
    times_used INT DEFAULT 0,
    acceptance_rate FLOAT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tod_challenges_type ON tod_challenges(type);
CREATE INDEX idx_tod_challenges_difficulty ON tod_challenges(difficulty);
CREATE INDEX idx_tod_challenges_category ON tod_challenges(category);
CREATE INDEX idx_tod_challenges_gender ON tod_challenges(gender_target);
CREATE INDEX idx_tod_challenges_active ON tod_challenges(is_active);

-- Games table
CREATE TABLE IF NOT EXISTS tod_games (
    id SERIAL PRIMARY KEY,
    match_id INT NOT NULL UNIQUE REFERENCES match_sessions(id) ON DELETE CASCADE,
    state VARCHAR(30) NOT NULL DEFAULT 'matchmaking',
    current_turn_id INT,
    active_player_id INT,
    passive_player_id INT,
    current_round INT DEFAULT 1,
    max_rounds INT DEFAULT 10,
    turn_started_at TIMESTAMP,
    turn_deadline TIMESTAMP,
    warning_shown_at TIMESTAMP,
    allow_items BOOLEAN DEFAULT true,
    difficulty_level VARCHAR(20) DEFAULT 'normal',
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    winner_id INT,
    end_reason VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tod_games_match ON tod_games(match_id);
CREATE INDEX idx_tod_games_state ON tod_games(state);
CREATE INDEX idx_tod_games_active_player ON tod_games(active_player_id);
CREATE INDEX idx_tod_games_passive_player ON tod_games(passive_player_id);
CREATE INDEX idx_tod_games_deadline ON tod_games(turn_deadline);
CREATE INDEX idx_tod_games_current_turn ON tod_games(current_turn_id);

-- Turns table
CREATE TABLE IF NOT EXISTS tod_turns (
    id SERIAL PRIMARY KEY,
    game_id INT NOT NULL REFERENCES tod_games(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    player_id INT NOT NULL,
    judge_id INT NOT NULL,
    choice VARCHAR(10),
    chosen_at TIMESTAMP,
    challenge_id INT REFERENCES tod_challenges(id),
    challenge_text TEXT,
    proof_type VARCHAR(20),
    proof_data TEXT,
    proof_submitted_at TIMESTAMP,
    judgment_result VARCHAR(20),
    judgment_reason TEXT,
    judged_at TIMESTAMP,
    item_used VARCHAR(20),
    item_used_at TIMESTAMP,
    xp_awarded INT DEFAULT 0,
    coins_awarded INT DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    timeout_at TIMESTAMP
);

CREATE INDEX idx_tod_turns_game ON tod_turns(game_id);
CREATE INDEX idx_tod_turns_player ON tod_turns(player_id);
CREATE INDEX idx_tod_turns_judge ON tod_turns(judge_id);
CREATE INDEX idx_tod_turns_round ON tod_turns(game_id, round_number);

-- Player stats table
CREATE TABLE IF NOT EXISTS tod_player_stats (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    games_played INT DEFAULT 0,
    games_won INT DEFAULT 0,
    games_lost INT DEFAULT 0,
    truths_chosen INT DEFAULT 0,
    dares_chosen INT DEFAULT 0,
    challenges_completed INT DEFAULT 0,
    challenges_failed INT DEFAULT 0,
    judgments_made INT DEFAULT 0,
    judgments_accepted INT DEFAULT 0,
    judgments_rejected INT DEFAULT 0,
    judge_score FLOAT DEFAULT 100.0,
    unfair_judgment_count INT DEFAULT 0,
    items_used INT DEFAULT 0,
    shields_owned INT DEFAULT 1,
    swaps_owned INT DEFAULT 1,
    mirrors_owned INT DEFAULT 1,
    avg_response_time INT DEFAULT 0,
    timeout_count INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tod_player_stats_user ON tod_player_stats(user_id);
CREATE INDEX idx_tod_player_stats_judge_score ON tod_player_stats(judge_score);

-- Judgment logs table
CREATE TABLE IF NOT EXISTS tod_judgment_logs (
    id SERIAL PRIMARY KEY,
    turn_id INT NOT NULL REFERENCES tod_turns(id) ON DELETE CASCADE,
    judge_id INT NOT NULL,
    player_id INT NOT NULL,
    result VARCHAR(20) NOT NULL,
    proof_quality INT DEFAULT 0,
    is_suspicious BOOLEAN DEFAULT false,
    suspicion_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tod_judgment_logs_turn ON tod_judgment_logs(turn_id);
CREATE INDEX idx_tod_judgment_logs_judge ON tod_judgment_logs(judge_id);
CREATE INDEX idx_tod_judgment_logs_player ON tod_judgment_logs(player_id);
CREATE INDEX idx_tod_judgment_logs_suspicious ON tod_judgment_logs(is_suspicious);
CREATE INDEX idx_tod_judgment_logs_judge_created ON tod_judgment_logs(judge_id, created_at DESC);

-- Action logs table for idempotency
CREATE TABLE IF NOT EXISTS tod_action_logs (
    id SERIAL PRIMARY KEY,
    game_id INT NOT NULL,
    user_id INT NOT NULL,
    action_id VARCHAR(100) NOT NULL UNIQUE,
    action VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tod_action_logs_game ON tod_action_logs(game_id);
CREATE INDEX idx_tod_action_logs_user ON tod_action_logs(user_id);
CREATE UNIQUE INDEX idx_tod_action_logs_action_id ON tod_action_logs(action_id);

-- Seed initial challenges
INSERT INTO tod_challenges (type, text, difficulty, category, gender_target, relation_level, proof_type, proof_hint, xp_reward, coin_reward) VALUES
-- Easy Truth Questions
('truth', 'آخرین باری که دروغ گفتی کی بود و چرا؟', 'easy', 'funny', 'all', 'stranger', 'text', 'پاسخ صادقانه بده', 15, 10),
('truth', 'بدترین خاطره‌ای که از مدرسه داری چیه؟', 'easy', 'funny', 'all', 'friend', 'text', 'یه خاطره واقعی تعریف کن', 15, 10),
('truth', 'اگه می‌تونستی یه روز با کسی جا عوض کنی، کی بود؟', 'easy', 'funny', 'all', 'stranger', 'text', 'اسم ببر و دلیلش رو بگو', 15, 10),
('truth', 'خجالت‌آورترین اتفاقی که برات افتاده چی بوده؟', 'easy', 'embarrassing', 'all', 'friend', 'text', 'یه داستان واقعی بگو', 18, 12),
('truth', 'اگه یه قدرت فوق‌العاده داشتی، چی انتخاب می‌کردی؟', 'easy', 'funny', 'all', 'stranger', 'text', 'قدرت و دلیلش رو بگو', 15, 10),

-- Medium Truth Questions
('truth', 'بزرگترین ترست که داری چیه؟', 'medium', 'embarrassing', 'all', 'friend', 'text', 'صادقانه بگو', 20, 15),
('truth', 'آخرین باری که گریه کردی کی بود و چرا؟', 'medium', 'romantic', 'all', 'close', 'text', 'داستانش رو بگو', 20, 15),
('truth', 'اگه فقط یه نفر رو می‌تونستی ببخشی، کی بود؟', 'medium', 'romantic', 'all', 'close', 'text', 'اسم و دلیل', 20, 15),

-- Easy Dare Challenges
('dare', 'یه ویس بفرست و بگو: من سلطان تنبلیام!', 'easy', 'funny', 'all', 'stranger', 'voice', 'ویس بفرست', 20, 15),
('dare', 'یه سلفی خنده‌دار از خودت بگیر و بفرست', 'easy', 'funny', 'all', 'friend', 'image', 'عکس بفرست', 22, 18),
('dare', 'یه ویس بفرست و مثل گربه میو کن!', 'easy', 'funny', 'all', 'stranger', 'voice', 'ویس بفرست', 20, 15),
('dare', 'یه عکس از آخرین چیزی که خوردی بفرست', 'easy', 'funny', 'all', 'stranger', 'image', 'عکس بفرست', 18, 12),
('dare', 'یه ویس بفرست و یه جوک بگو', 'easy', 'funny', 'all', 'friend', 'voice', 'ویس بفرست', 20, 15),

-- Medium Dare Challenges
('dare', 'یه ویدیو کوتاه از خودت برقص و بفرست', 'medium', 'embarrassing', 'all', 'close', 'video', 'ویدیو بفرست', 30, 25),
('dare', 'یه عکس سلفی با یه حالت عجیب بگیر', 'medium', 'embarrassing', 'all', 'friend', 'image', 'عکس بفرست', 25, 20),
('dare', 'یه ویس بفرست و یه آهنگ بخون', 'medium', 'funny', 'all', 'friend', 'voice', 'ویس بفرست', 25, 20),

-- Hard Challenges
('dare', 'یه ویدیو از خودت بفرست که داری یه کار خنده‌دار انجام میدی', 'hard', 'embarrassing', 'all', 'close', 'video', 'ویدیو بفرست', 35, 30),
('truth', 'بزرگترین رازی که از کسی پنهون کردی چیه؟', 'hard', 'romantic', 'all', 'close', 'text', 'صادقانه بگو', 30, 25);

COMMENT ON TABLE tod_games IS 'Truth or Dare game sessions';
COMMENT ON TABLE tod_challenges IS 'Truth or Dare challenge questions';
COMMENT ON TABLE tod_turns IS 'Individual turns in ToD games';
COMMENT ON TABLE tod_player_stats IS 'Player statistics for ToD games';
COMMENT ON TABLE tod_judgment_logs IS 'Judgment history for anti-abuse';
COMMENT ON TABLE tod_action_logs IS 'Action deduplication logs';
