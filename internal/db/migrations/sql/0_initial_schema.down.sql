-- File: Rollback Initial Database Schema

DROP TABLE IF EXISTS link_file_projection CASCADE;
DROP TABLE IF EXISTS link_projection CASCADE;
DROP TABLE IF EXISTS link_event CASCADE;
DROP TABLE IF EXISTS file_permission_projection CASCADE;
DROP TABLE IF EXISTS file_projection CASCADE;
DROP TABLE IF EXISTS user_account_event CASCADE;
DROP TABLE IF EXISTS file_event CASCADE;
DROP TABLE IF EXISTS user_account_email_projection CASCADE;
DROP TABLE IF EXISTS user_account_mfa_recovery_code_projection CASCADE;
DROP TABLE IF EXISTS user_account_projection CASCADE;

DROP TYPE IF EXISTS file_permission_type CASCADE;
