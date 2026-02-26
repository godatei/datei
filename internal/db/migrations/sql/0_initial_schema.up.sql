-- Datei: Initial Database Schema
-- High performance self-hosted document management solution

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
-- Datei (Core Table)
-- ============================================================================

CREATE TABLE datei (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  parent_id UUID REFERENCES datei(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_datei_id UUID REFERENCES datei(id) ON DELETE SET NULL,
  latest_name_id UUID,
  latest_version_id UUID,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  trashed_at TIMESTAMPTZ,
  trashed_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_datei_parent_id ON datei(parent_id);
CREATE INDEX idx_datei_linked_datei_id ON datei(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_datei_trashed_at ON datei(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_datei_created_by ON datei(created_by) WHERE created_by IS NOT NULL;

-- ============================================================================
-- Datei Name (Name History)
-- ============================================================================

CREATE TABLE datei_name (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_datei_name_datei_id ON datei_name(datei_id);

-- Deferred FK: datei.latest_name_id -> datei_name.id
ALTER TABLE datei
  ADD CONSTRAINT fk_datei_latest_name
  FOREIGN KEY (latest_name_id) REFERENCES datei_name(id) ON DELETE RESTRICT;

CREATE INDEX idx_datei_latest_name_id ON datei(latest_name_id) WHERE latest_name_id IS NOT NULL;

-- ============================================================================
-- Datei Version
-- ============================================================================

CREATE TABLE datei_version (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  s3_key TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  checksum TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  content_md TEXT,
  content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content_md, ''))) STORED,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_datei_version_datei_id ON datei_version(datei_id);
CREATE INDEX idx_datei_version_content_search ON datei_version USING GIN(content_search);

-- Deferred FK: datei.latest_version_id -> datei_version.id
ALTER TABLE datei
  ADD CONSTRAINT fk_datei_latest_version
  FOREIGN KEY (latest_version_id) REFERENCES datei_version(id) ON DELETE RESTRICT;

CREATE INDEX idx_datei_latest_version_id ON datei(latest_version_id) WHERE latest_version_id IS NOT NULL;

-- ============================================================================
-- Datei Label (Relation Table)
-- ============================================================================

CREATE TABLE datei_label (
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  label_id UUID NOT NULL REFERENCES label(id) ON DELETE CASCADE,
  PRIMARY KEY (datei_id, label_id)
);

CREATE INDEX idx_datei_label_label_id ON datei_label(label_id);

-- ============================================================================
-- Datei Annotation (Key-Value Pairs)
-- ============================================================================

CREATE TABLE datei_annotation (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (datei_id, key)
);

CREATE INDEX idx_datei_annotation_datei_id ON datei_annotation(datei_id);

-- ============================================================================
-- Datei Permission
-- ============================================================================

CREATE TYPE datei_permission_type AS ENUM ('owner', 'read_write', 'read_only');

CREATE TABLE datei_permission (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
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
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  PRIMARY KEY (public_link_id, datei_id)
);

CREATE INDEX idx_public_link_datei_datei_id ON public_link_datei(datei_id);

-- ============================================================================
-- Datei Comment
-- ============================================================================

CREATE TABLE datei_comment (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES datei(id) ON DELETE CASCADE,
  user_account_id UUID NOT NULL REFERENCES user_account(id) ON DELETE RESTRICT,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_datei_comment_datei_id ON datei_comment(datei_id);
CREATE INDEX idx_datei_comment_user_account_id ON datei_comment(user_account_id);

-- ============================================================================
-- Audit Log
-- ============================================================================

CREATE TABLE audit_log (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  actor_id UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID NOT NULL,
  metadata JSONB,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_actor_id ON audit_log(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_audit_log_target ON audit_log(target_type, target_id);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
