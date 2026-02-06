-- Migration: Drop Truth or Dare tables
-- File: scripts/migrations/006_create_tod_tables.down.sql

DROP TABLE IF EXISTS tod_action_logs CASCADE;
DROP TABLE IF EXISTS tod_judgment_logs CASCADE;
DROP TABLE IF EXISTS tod_player_stats CASCADE;
DROP TABLE IF EXISTS tod_turns CASCADE;
DROP TABLE IF EXISTS tod_games CASCADE;
DROP TABLE IF EXISTS tod_challenges CASCADE;
