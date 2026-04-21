-- Datei: Initial Database Schema

-- ============================================================================
-- User Account Projections
-- ============================================================================

CREATE TABLE user_account_projection (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL,
  password_hash BYTEA NOT NULL,
  password_salt BYTEA NOT NULL,
  mfa_secret TEXT,
  mfa_enabled BOOLEAN NOT NULL DEFAULT false,
  mfa_enabled_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  last_logged_in_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_user_account_projection_mfa CHECK (
    mfa_enabled = false OR mfa_secret IS NOT NULL
  )
);

CREATE INDEX idx_user_account_projection_archived_at ON user_account_projection(archived_at) WHERE archived_at IS NOT NULL;

CREATE TABLE user_account_mfa_recovery_code_projection (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES user_account_projection(id) ON DELETE CASCADE,
  code_hash BYTEA NOT NULL,
  code_salt BYTEA NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_account_mfa_recovery_code_projection_user_account_id ON user_account_mfa_recovery_code_projection(user_account_id, used_at);

CREATE TABLE user_account_email_projection (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES user_account_projection(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  verified_at TIMESTAMPTZ,
  is_primary BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_account_email_projection_user_account_id ON user_account_email_projection(user_account_id);
CREATE UNIQUE INDEX uq_user_account_email_projection_primary ON user_account_email_projection(user_account_id) WHERE is_primary = true;

-- ============================================================================
-- Event Stores
-- ============================================================================

CREATE TABLE datei_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_event_store_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_datei_event_stream_id ON datei_event(stream_id);
CREATE INDEX idx_datei_event_created_at ON datei_event(created_at DESC);
CREATE INDEX idx_datei_event_event_type ON datei_event(event_type);

CREATE TABLE user_account_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_user_account_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_user_account_event_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_user_account_event_stream_id ON user_account_event(stream_id);
CREATE INDEX idx_user_account_event_created_at ON user_account_event(created_at DESC);
CREATE INDEX idx_user_account_event_event_type ON user_account_event(event_type);

-- ============================================================================
-- Datei Permission Type
-- ============================================================================

CREATE TYPE datei_permission_type AS ENUM ('owner', 'read_write', 'read_only');

-- ============================================================================
-- Datei Projection
-- ============================================================================

CREATE TABLE datei_projection (
  id UUID PRIMARY KEY,
  parent_id UUID REFERENCES datei_projection(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_datei_id UUID REFERENCES datei_projection(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  s3_key TEXT,
  size BIGINT,
  checksum TEXT,
  mime_type TEXT,
  content_md TEXT,
  content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content_md, ''))) STORED,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  trashed_at TIMESTAMPTZ,
  created_by UUID REFERENCES user_account_projection(id) ON DELETE RESTRICT,
  updated_by UUID REFERENCES user_account_projection(id) ON DELETE RESTRICT,
  trashed_by UUID REFERENCES user_account_projection(id) ON DELETE RESTRICT
);

CREATE INDEX idx_datei_projection_parent_id ON datei_projection(parent_id);
CREATE INDEX idx_datei_projection_linked_datei_id ON datei_projection(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_datei_projection_content_search ON datei_projection USING GIN(content_search);

-- ============================================================================
-- Datei Permission Projection
--
-- user_group_id has no FK: the user_group table is not yet event-sourced and
-- has been removed from this migration. The column is retained because
-- DateiPermissionGranted/Revoked events still carry UserGroupID for forward
-- compatibility with a future groups domain.
-- ============================================================================

CREATE TABLE datei_permission_projection (
  id UUID PRIMARY KEY,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES user_account_projection(id) ON DELETE RESTRICT,
  user_group_id UUID,
  permission_type datei_permission_type NOT NULL,
  is_favorite BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ck_datei_permission_projection_grantee CHECK (
    (user_account_id IS NOT NULL AND user_group_id IS NULL) OR
    (user_account_id IS NULL AND user_group_id IS NOT NULL)
  )
);

CREATE INDEX idx_datei_permission_projection_datei_id ON datei_permission_projection(datei_id);
CREATE INDEX idx_datei_permission_projection_user_account_id ON datei_permission_projection(user_account_id) WHERE user_account_id IS NOT NULL;
CREATE INDEX idx_datei_permission_projection_user_group_id ON datei_permission_projection(user_group_id) WHERE user_group_id IS NOT NULL;
CREATE UNIQUE INDEX uq_datei_permission_projection_owner ON datei_permission_projection(datei_id) WHERE permission_type = 'owner';
CREATE UNIQUE INDEX uq_datei_permission_projection_user ON datei_permission_projection(datei_id, user_account_id) WHERE user_account_id IS NOT NULL;
CREATE UNIQUE INDEX uq_datei_permission_projection_group ON datei_permission_projection(datei_id, user_group_id) WHERE user_group_id IS NOT NULL;
