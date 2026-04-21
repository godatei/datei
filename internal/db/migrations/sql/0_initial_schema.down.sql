-- Datei: Rollback Initial Database Schema

DROP TABLE IF EXISTS datei_permission_projection CASCADE;
DROP TABLE IF EXISTS datei_projection CASCADE;
DROP TABLE IF EXISTS user_account_event CASCADE;
DROP TABLE IF EXISTS datei_event CASCADE;
DROP TABLE IF EXISTS user_account_email_projection CASCADE;
DROP TABLE IF EXISTS user_account_mfa_recovery_code_projection CASCADE;
DROP TABLE IF EXISTS user_account_projection CASCADE;

DROP TYPE IF EXISTS datei_permission_type CASCADE;
