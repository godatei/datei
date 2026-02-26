-- Datei: Rollback Initial Database Schema

-- ============================================================================
-- Drop Tables (in reverse dependency order)
-- ============================================================================

DROP TABLE IF EXISTS audit_log CASCADE;

DROP TABLE IF EXISTS datei_comment CASCADE;

DROP TABLE IF EXISTS public_link_datei CASCADE;

DROP TABLE IF EXISTS public_link CASCADE;

DROP TABLE IF EXISTS datei_permission CASCADE;

DROP TABLE IF EXISTS datei_annotation CASCADE;

DROP TABLE IF EXISTS datei_label CASCADE;

DROP TABLE IF EXISTS datei_version CASCADE;

DROP TABLE IF EXISTS datei_name CASCADE;

DROP TABLE IF EXISTS datei CASCADE;

DROP TABLE IF EXISTS label CASCADE;

DROP TABLE IF EXISTS user_group_member CASCADE;

DROP TABLE IF EXISTS user_group CASCADE;

DROP TABLE IF EXISTS user_email CASCADE;

DROP TABLE IF EXISTS user_account_mfa_recovery_code CASCADE;

DROP TABLE IF EXISTS user_account CASCADE;

-- ============================================================================
-- Drop Custom Types (ENUMs)
-- ============================================================================

DROP TYPE IF EXISTS public_link_permission_type CASCADE;

DROP TYPE IF EXISTS datei_permission_type CASCADE;

DROP TYPE IF EXISTS user_group_role CASCADE;
