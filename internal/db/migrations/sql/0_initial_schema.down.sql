-- Datei: Rollback Initial Database Schema

DROP TABLE IF EXISTS datei_comment CASCADE;
DROP TABLE IF EXISTS public_link_datei CASCADE;
DROP TABLE IF EXISTS public_link CASCADE;
DROP TABLE IF EXISTS datei_permission CASCADE;
DROP TABLE IF EXISTS datei_annotation CASCADE;
DROP TABLE IF EXISTS datei_label CASCADE;
DROP TABLE IF EXISTS datei_permission_projection CASCADE;
DROP TABLE IF EXISTS datei_projection CASCADE;
DROP TABLE IF EXISTS user_account_event CASCADE;
DROP TABLE IF EXISTS datei_event CASCADE;
DROP TABLE IF EXISTS label CASCADE;
DROP TABLE IF EXISTS user_group_member CASCADE;
DROP TABLE IF EXISTS user_group CASCADE;
DROP TABLE IF EXISTS user_account_email_projection CASCADE;
DROP TABLE IF EXISTS user_account_mfa_recovery_code_projection CASCADE;
DROP TABLE IF EXISTS user_account_projection CASCADE;

DROP TYPE IF EXISTS public_link_permission_type CASCADE;
DROP TYPE IF EXISTS datei_permission_type CASCADE;
DROP TYPE IF EXISTS user_group_role CASCADE;
