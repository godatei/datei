-- Datei: Initial Database Schema

-- ============================================================================
-- User & Group Tables
-- ============================================================================

CREATE TABLE user_account (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  password_salt TEXT NOT NULL,
  mfa_secret TEXT,
  mfa_enabled BOOLEAN NOT NULL DEFAULT false,
  mfa_enabled_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_user_account_mfa CHECK (
    mfa_enabled = false OR mfa_secret IS NOT NULL
  )
);

CREATE INDEX idx_user_account_archived_at ON user_account(archived_at) WHERE archived_at IS NOT NULL;

CREATE TABLE user_account_mfa_recovery_code (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  code_salt TEXT NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_account_mfa_recovery_code_user_account_id ON user_account_mfa_recovery_code(user_account_id, used_at);

CREATE TABLE user_email (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  verified_at TIMESTAMPTZ,
  is_primary BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_email_user_account_id ON user_email(user_account_id);
CREATE UNIQUE INDEX uq_user_email_primary ON user_email(user_account_id) WHERE is_primary = true;

CREATE TABLE user_group (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_group_created_by ON user_group(created_by);
CREATE INDEX idx_user_group_archived_at ON user_group(archived_at) WHERE archived_at IS NOT NULL;

CREATE TYPE user_group_role AS ENUM ('admin', 'member');

CREATE TABLE user_group_member (
  user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
  user_group_id UUID NOT NULL REFERENCES user_group(id) ON DELETE RESTRICT,
  role user_group_role NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_account_id, user_group_id)
);

CREATE INDEX idx_user_group_member_user_group_id ON user_group_member(user_group_id);

-- ============================================================================
-- Label Table
-- ============================================================================

CREATE TABLE label (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  foreground_color TEXT NOT NULL DEFAULT '#FFFFFF',
  background_color TEXT NOT NULL DEFAULT '#000000',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- Event Store
-- ============================================================================

CREATE TABLE event_store (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  event_metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_event_store_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_event_store_stream_id ON event_store(stream_id);
CREATE INDEX idx_event_store_created_at ON event_store(created_at DESC);
CREATE INDEX idx_event_store_event_type ON event_store(event_type);

-- ============================================================================
-- Datei Permission Type
-- ============================================================================

CREATE TYPE datei_permission_type AS ENUM ('owner', 'read_write', 'read_only');

-- ============================================================================
-- Datei Projection (Current state — read model updated by event handlers)
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
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  trashed_at TIMESTAMPTZ,
  trashed_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  projection_version INT NOT NULL DEFAULT 1
);

CREATE INDEX idx_datei_projection_parent_id ON datei_projection(parent_id);
CREATE INDEX idx_datei_projection_linked_datei_id ON datei_projection(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_datei_projection_trashed_at ON datei_projection(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_datei_projection_created_by ON datei_projection(created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_datei_projection_content_search ON datei_projection USING GIN(content_search);

-- ============================================================================
-- Datei Permission Projection (Access control — read model)
-- ============================================================================

CREATE TABLE datei_permission_projection (
  id UUID PRIMARY KEY,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  user_group_id UUID REFERENCES user_group(id) ON DELETE RESTRICT,
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

-- ============================================================================
-- Datei Label (Relation Table)
-- ============================================================================

CREATE TABLE datei_label (
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  label_id UUID NOT NULL REFERENCES label(id) ON DELETE CASCADE,
  PRIMARY KEY (datei_id, label_id)
);

CREATE INDEX idx_datei_label_label_id ON datei_label(label_id);

-- ============================================================================
-- Datei Annotation (Key-Value Pairs)
-- ============================================================================

CREATE TABLE datei_annotation (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (datei_id, key)
);

CREATE INDEX idx_datei_annotation_datei_id ON datei_annotation(datei_id);

-- ============================================================================
-- Datei Permission (Write model)
-- ============================================================================

CREATE TABLE datei_permission (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  user_group_id UUID REFERENCES user_group(id) ON DELETE RESTRICT,
  permission_type datei_permission_type NOT NULL,
  is_favorite BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_datei_permission_grantee CHECK (
    (user_account_id IS NOT NULL AND user_group_id IS NULL) OR
    (user_account_id IS NULL AND user_group_id IS NOT NULL)
  )
);

CREATE INDEX idx_datei_permission_datei_id ON datei_permission(datei_id);
CREATE INDEX idx_datei_permission_user_account_id ON datei_permission(user_account_id) WHERE user_account_id IS NOT NULL;
CREATE INDEX idx_datei_permission_user_group_id ON datei_permission(user_group_id) WHERE user_group_id IS NOT NULL;
CREATE UNIQUE INDEX uq_datei_permission_owner ON datei_permission(datei_id) WHERE permission_type = 'owner';
CREATE UNIQUE INDEX uq_datei_permission_user ON datei_permission(datei_id, user_account_id) WHERE user_account_id IS NOT NULL;
CREATE UNIQUE INDEX uq_datei_permission_group ON datei_permission(datei_id, user_group_id) WHERE user_group_id IS NOT NULL;

-- ============================================================================
-- Public Link
-- ============================================================================

CREATE TYPE public_link_permission_type AS ENUM ('read_only', 'read_write');

CREATE TABLE public_link (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  token TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
  permission_type public_link_permission_type NOT NULL DEFAULT 'read_only',
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_public_link_created_by ON public_link(created_by);
CREATE INDEX idx_public_link_expires_at ON public_link(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- Public Link Datei (Relation Table)
-- ============================================================================

CREATE TABLE public_link_datei (
  public_link_id UUID NOT NULL REFERENCES public_link(id) ON DELETE CASCADE,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  PRIMARY KEY (public_link_id, datei_id)
);

CREATE INDEX idx_public_link_datei_datei_id ON public_link_datei(datei_id);

-- ============================================================================
-- Datei Comment
-- ============================================================================

CREATE TABLE datei_comment (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_datei_comment_datei_id ON datei_comment(datei_id);
CREATE INDEX idx_datei_comment_user_account_id ON datei_comment(user_account_id);
