-- File: Initial Database Schema

-- ============================================================================
-- User Account Projections
-- ============================================================================

CREATE TABLE user_account_projection (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL,
  password_hash BYTEA NOT NULL,
  password_salt BYTEA NOT NULL,
  is_admin BOOLEAN NOT NULL DEFAULT true,
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

CREATE TABLE file_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_event_store_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_file_event_stream_id ON file_event(stream_id);
CREATE INDEX idx_file_event_created_at ON file_event(created_at DESC);
CREATE INDEX idx_file_event_event_type ON file_event(event_type);

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
-- File Permission Type
-- ============================================================================

CREATE TYPE file_permission_type AS ENUM ('owner', 'read_write', 'read_only');

-- ============================================================================
-- File Projection
-- ============================================================================

CREATE TABLE file_projection (
  id UUID PRIMARY KEY,
  parent_id UUID REFERENCES file_projection(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_file_id UUID REFERENCES file_projection(id) ON DELETE SET NULL,
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

CREATE INDEX idx_file_projection_parent_id ON file_projection(parent_id);
CREATE INDEX idx_file_projection_linked_file_id ON file_projection(linked_file_id) WHERE linked_file_id IS NOT NULL;
CREATE INDEX idx_file_projection_content_search ON file_projection USING GIN(content_search);

-- ============================================================================
-- File Permission Projection
--
-- user_group_id has no FK: the user_group table is not yet event-sourced and
-- has been removed from this migration. The column is retained because
-- FilePermissionGranted/Revoked events still carry UserGroupID for forward
-- compatibility with a future groups domain.
-- ============================================================================

CREATE TABLE file_permission_projection (
  id UUID PRIMARY KEY,
  file_id UUID NOT NULL REFERENCES file_projection(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES user_account_projection(id) ON DELETE RESTRICT,
  user_group_id UUID,
  permission_type file_permission_type NOT NULL,
  is_favorite BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ck_file_permission_projection_grantee CHECK (
    (user_account_id IS NOT NULL AND user_group_id IS NULL) OR
    (user_account_id IS NULL AND user_group_id IS NOT NULL)
  )
);

CREATE INDEX idx_file_permission_projection_file_id ON file_permission_projection(file_id);
CREATE INDEX idx_file_permission_projection_user_account_id ON file_permission_projection(user_account_id) WHERE user_account_id IS NOT NULL;
CREATE INDEX idx_file_permission_projection_user_group_id ON file_permission_projection(user_group_id) WHERE user_group_id IS NOT NULL;
CREATE UNIQUE INDEX uq_file_permission_projection_owner ON file_permission_projection(file_id) WHERE permission_type = 'owner';
CREATE UNIQUE INDEX uq_file_permission_projection_user ON file_permission_projection(file_id, user_account_id) WHERE user_account_id IS NOT NULL;
CREATE UNIQUE INDEX uq_file_permission_projection_group ON file_permission_projection(file_id, user_group_id) WHERE user_group_id IS NOT NULL;

-- ============================================================================
-- Link Event Store (public sharing domain)
-- ============================================================================

CREATE TABLE link_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_link_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_link_event_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_link_event_stream_id ON link_event(stream_id);
CREATE INDEX idx_link_event_created_at ON link_event(created_at DESC);
CREATE INDEX idx_link_event_event_type ON link_event(event_type);

-- ============================================================================
-- Link Projection
-- ============================================================================

CREATE TABLE link_projection (
  id UUID PRIMARY KEY,
  owner_id UUID NOT NULL REFERENCES user_account_projection(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  key TEXT NOT NULL UNIQUE,
  -- Plain-text shared secret (intentional; the value is displayed back to the
  -- owner so they can share it with viewers out-of-band).
  code TEXT,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  open_count BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ck_link_projection_name_length CHECK (length(name) BETWEEN 1 AND 255),
  CONSTRAINT ck_link_projection_code_length CHECK (code IS NULL OR length(code) BETWEEN 1 AND 128)
);

CREATE INDEX idx_link_projection_owner_id ON link_projection(owner_id);

-- ============================================================================
-- Link <-> File Join Projection
-- ============================================================================

CREATE TABLE link_file_projection (
  link_id UUID NOT NULL REFERENCES link_projection(id) ON DELETE CASCADE,
  file_id UUID NOT NULL REFERENCES file_projection(id) ON DELETE CASCADE,
  added_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (link_id, file_id)
);

CREATE INDEX idx_link_file_projection_file_id ON link_file_projection(file_id);
