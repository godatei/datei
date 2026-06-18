-- File: Target schema (pre-event-sourcing design).
-- High performance self-hosted document management solution.
--
-- Status (2026-04-21): design does not yet include event sourcing.
-- This file is kept for reference only. The live migration
-- (internal/db/migrations/sql/0_initial_schema.up.sql) provisions a reduced
-- subset consisting of event stores and projections for event-sourced
-- domains. Non-ES tables below (user_group*, label, file_label,
-- file_annotation, file_permission write model, public_link*, file_comment)
-- will be reintroduced via per-domain event sourcing as those domains land.

-- ============================================================================
-- User & Group Tables
-- ============================================================================

CREATE TABLE UserAccount (
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
  CONSTRAINT ck_UserAccount_mfa CHECK (
    mfa_enabled = false OR mfa_secret IS NOT NULL
  )
);

CREATE INDEX idx_UserAccount_archived_at ON UserAccount(archived_at) WHERE archived_at IS NOT NULL;

CREATE TABLE UserAccount_MFARecoveryCode (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  code_salt TEXT NOT NULL,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_UserAccount_MFARecoveryCode_user_account_id ON UserAccount_MFARecoveryCode(user_account_id, used_at);

CREATE TABLE UserAccountEmail (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  verified_at TIMESTAMPTZ,
  is_primary BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_UserAccountEmail_user_account_id ON UserAccountEmail(user_account_id);
CREATE UNIQUE INDEX uq_UserAccountEmail_primary ON UserAccountEmail(user_account_id) WHERE is_primary = true;

CREATE TABLE UserGroup (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES UserAccount(id) ON DELETE RESTRICT,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_UserGroup_created_by ON UserGroup(created_by);
CREATE INDEX idx_UserGroup_archived_at ON UserGroup(archived_at) WHERE archived_at IS NOT NULL;

CREATE TYPE UserGroupRole AS ENUM ('admin', 'member');

CREATE TABLE UserGroup_Member (
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE RESTRICT,
  user_group_id UUID NOT NULL REFERENCES UserGroup(id) ON DELETE RESTRICT,
  role UserGroupRole NOT NULL DEFAULT 'member',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_account_id, user_group_id)
);

CREATE INDEX idx_UserGroup_Member_user_group_id ON UserGroup_Member(user_group_id);

-- ============================================================================
-- Label Table
-- ============================================================================

CREATE TABLE Label (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  foreground_color TEXT NOT NULL DEFAULT '#FFFFFF',
  background_color TEXT NOT NULL DEFAULT '#000000',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- File (Core Table)
-- ============================================================================

CREATE TABLE File (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  parent_id UUID REFERENCES File(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_file_id UUID REFERENCES File(id) ON DELETE SET NULL,
  latest_name_id UUID,
  latest_version_id UUID,
  created_by UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  trashed_at TIMESTAMPTZ,
  trashed_by UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_File_parent_id ON File(parent_id);
CREATE INDEX idx_File_linked_file_id ON File(linked_file_id) WHERE linked_file_id IS NOT NULL;
CREATE INDEX idx_File_trashed_at ON File(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_File_created_by ON File(created_by) WHERE created_by IS NOT NULL;

-- ============================================================================
-- File Name (Name History)
-- ============================================================================

CREATE TABLE FileName (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_by UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_FileName_file_id ON FileName(file_id);

-- Deferred FK: File.latest_name_id -> FileName.id
ALTER TABLE File
  ADD CONSTRAINT fk_File_latest_name
  FOREIGN KEY (latest_name_id) REFERENCES FileName(id) ON DELETE RESTRICT;

CREATE INDEX idx_File_latest_name_id ON File(latest_name_id) WHERE latest_name_id IS NOT NULL;

-- ============================================================================
-- File Version
-- ============================================================================

CREATE TABLE FileVersion (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  version_number INTEGER NOT NULL,
  s3_bucket TEXT NOT NULL,
  s3_key TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  checksum TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  content_md TEXT,
  content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content_md, ''))) STORED,
  created_by UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (file_id, version_number)
);

CREATE INDEX idx_FileVersion_file_id ON FileVersion(file_id);
CREATE INDEX idx_FileVersion_content_search ON FileVersion USING GIN(content_search);

-- Deferred FK: File.latest_version_id -> FileVersion.id
ALTER TABLE File
  ADD CONSTRAINT fk_File_latest_version
  FOREIGN KEY (latest_version_id) REFERENCES FileVersion(id) ON DELETE RESTRICT;

CREATE INDEX idx_File_latest_version_id ON File(latest_version_id) WHERE latest_version_id IS NOT NULL;

-- ============================================================================
-- File Label (Relation Table)
-- ============================================================================

CREATE TABLE File_Label (
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  label_id UUID NOT NULL REFERENCES Label(id) ON DELETE CASCADE,
  PRIMARY KEY (file_id, label_id)
);

CREATE INDEX idx_File_Label_label_id ON File_Label(label_id);

-- ============================================================================
-- File Annotation (Key-Value Pairs)
-- ============================================================================

CREATE TABLE FileAnnotation (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (file_id, key)
);

CREATE INDEX idx_FileAnnotation_file_id ON FileAnnotation(file_id);

-- ============================================================================
-- File Permission
-- ============================================================================

CREATE TYPE FilePermissionType AS ENUM ('owner', 'read_write', 'read_only');

CREATE TABLE FilePermission (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  user_group_id UUID REFERENCES UserGroup(id) ON DELETE RESTRICT,
  permission_type FilePermissionType NOT NULL,
  is_favorite BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_FilePermission_grantee CHECK (
    (user_account_id IS NOT NULL AND user_group_id IS NULL) OR
    (user_account_id IS NULL AND user_group_id IS NOT NULL)
  )
);

CREATE INDEX idx_FilePermission_file_id ON FilePermission(file_id);
CREATE INDEX idx_FilePermission_user_account_id ON FilePermission(user_account_id) WHERE user_account_id IS NOT NULL;
CREATE INDEX idx_FilePermission_user_group_id ON FilePermission(user_group_id) WHERE user_group_id IS NOT NULL;
CREATE UNIQUE INDEX uq_FilePermission_owner ON FilePermission(file_id) WHERE permission_type = 'owner';
CREATE UNIQUE INDEX uq_FilePermission_user ON FilePermission(file_id, user_account_id) WHERE user_account_id IS NOT NULL;
CREATE UNIQUE INDEX uq_FilePermission_group ON FilePermission(file_id, user_group_id) WHERE user_group_id IS NOT NULL;

-- ============================================================================
-- Public Link
-- ============================================================================

CREATE TYPE PublicLinkPermissionType AS ENUM ('read_only', 'read_write');

CREATE TABLE PublicLink (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  token TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES UserAccount(id) ON DELETE RESTRICT,
  permission_type PublicLinkPermissionType NOT NULL DEFAULT 'read_only',
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_PublicLink_created_by ON PublicLink(created_by);
CREATE INDEX idx_PublicLink_expires_at ON PublicLink(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- Public Link File (Relation Table)
-- ============================================================================

CREATE TABLE PublicLink_File (
  public_link_id UUID NOT NULL REFERENCES PublicLink(id) ON DELETE CASCADE,
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  PRIMARY KEY (public_link_id, file_id)
);

CREATE INDEX idx_PublicLink_File_file_id ON PublicLink_File(file_id);

-- ============================================================================
-- File Comment
-- ============================================================================

CREATE TABLE FileComment (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  file_id UUID NOT NULL REFERENCES File(id) ON DELETE CASCADE,
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE RESTRICT,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_FileComment_file_id ON FileComment(file_id);
CREATE INDEX idx_FileComment_user_account_id ON FileComment(user_account_id);

-- ============================================================================
-- Audit Log
-- ============================================================================

CREATE TABLE AuditLog (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  actor_id UUID REFERENCES UserAccount(id) ON DELETE RESTRICT,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID NOT NULL,
  metadata JSONB,
  ip_address TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_AuditLog_actor_id ON AuditLog(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_AuditLog_target ON AuditLog(target_type, target_id);
CREATE INDEX idx_AuditLog_action ON AuditLog(action);
CREATE INDEX idx_AuditLog_created_at ON AuditLog(created_at);
