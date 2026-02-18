-- Datei: Initial Database Schema
-- High performance self-hosted document management solution

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
  mfa_enabled_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  CONSTRAINT ck_UserAccount_mfa CHECK (
    mfa_enabled = false OR mfa_secret IS NOT NULL
  )
);

CREATE TABLE UserAccount_MFARecoveryCode (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  code_salt TEXT NOT NULL,
  used_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_UserAccount_MFARecoveryCode_user_account_id ON UserAccount_MFARecoveryCode(user_account_id, used_at);

CREATE TABLE UserEmail (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  email TEXT NOT NULL UNIQUE,
  verified_at TIMESTAMP,
  is_primary BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_UserEmail_user_account_id ON UserEmail(user_account_id);
CREATE UNIQUE INDEX uq_UserEmail_primary ON UserEmail(user_account_id) WHERE is_primary = true;

CREATE TABLE UserGroup (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_UserGroup_created_by ON UserGroup(created_by);

CREATE TYPE UserGroupRole AS ENUM ('admin', 'member');

CREATE TABLE UserGroup_Member (
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  user_group_id UUID NOT NULL REFERENCES UserGroup(id) ON DELETE CASCADE,
  role UserGroupRole NOT NULL DEFAULT 'member',
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
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
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

-- ============================================================================
-- Datei (Core Table)
-- ============================================================================

CREATE TABLE Datei (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  parent_id UUID REFERENCES Datei(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  original_filename TEXT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  mime_type TEXT,
  file_size BIGINT,
  s3_bucket TEXT,
  s3_key TEXT,
  checksum TEXT,
  linked_datei_id UUID REFERENCES Datei(id) ON DELETE SET NULL,
  latest_version_id UUID,
  content_md TEXT,
  content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content_md, ''))) STORED,
  created_by UUID REFERENCES UserAccount(id) ON DELETE SET NULL,
  trashed_at TIMESTAMP,
  trashed_by UUID REFERENCES UserAccount(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_Datei_parent_id ON Datei(parent_id);
CREATE INDEX idx_Datei_linked_datei_id ON Datei(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_Datei_trashed_at ON Datei(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_Datei_content_search ON Datei USING GIN(content_search);
CREATE INDEX idx_Datei_created_by ON Datei(created_by) WHERE created_by IS NOT NULL;

-- ============================================================================
-- Datei Version
-- ============================================================================

CREATE TABLE DateiVersion (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  version_number INTEGER NOT NULL,
  s3_bucket TEXT NOT NULL,
  s3_key TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  checksum TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  content_md TEXT,
  created_by UUID REFERENCES UserAccount(id) ON DELETE SET NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  UNIQUE (datei_id, version_number)
);

CREATE INDEX idx_DateiVersion_datei_id ON DateiVersion(datei_id);

-- Deferred FK: Datei.latest_version_id -> DateiVersion.id
ALTER TABLE Datei
  ADD CONSTRAINT fk_Datei_latest_version
  FOREIGN KEY (latest_version_id) REFERENCES DateiVersion(id) ON DELETE SET NULL;

CREATE INDEX idx_Datei_latest_version_id ON Datei(latest_version_id) WHERE latest_version_id IS NOT NULL;

-- ============================================================================
-- Datei Label (Relation Table)
-- ============================================================================

CREATE TABLE Datei_Label (
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  label_id UUID NOT NULL REFERENCES Label(id) ON DELETE CASCADE,
  PRIMARY KEY (datei_id, label_id)
);

CREATE INDEX idx_Datei_Label_label_id ON Datei_Label(label_id);

-- ============================================================================
-- Datei Annotation (Key-Value Pairs)
-- ============================================================================

CREATE TABLE DateiAnnotation (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  UNIQUE (datei_id, key)
);

CREATE INDEX idx_DateiAnnotation_datei_id ON DateiAnnotation(datei_id);

-- ============================================================================
-- Datei Permission
-- ============================================================================

CREATE TABLE DateiPermission (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  user_account_id UUID REFERENCES UserAccount(id) ON DELETE CASCADE,
  user_group_id UUID REFERENCES UserGroup(id) ON DELETE CASCADE,
  permission_type TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  CONSTRAINT ck_DateiPermission_grantee CHECK (
    (user_account_id IS NOT NULL AND user_group_id IS NULL) OR
    (user_account_id IS NULL AND user_group_id IS NOT NULL)
  ),
  CONSTRAINT ck_DateiPermission_type CHECK (
    permission_type IN ('owner', 'read_write', 'read_only')
  )
);

CREATE INDEX idx_DateiPermission_datei_id ON DateiPermission(datei_id);
CREATE INDEX idx_DateiPermission_user_account_id ON DateiPermission(user_account_id) WHERE user_account_id IS NOT NULL;
CREATE INDEX idx_DateiPermission_user_group_id ON DateiPermission(user_group_id) WHERE user_group_id IS NOT NULL;
CREATE UNIQUE INDEX uq_DateiPermission_owner ON DateiPermission(datei_id) WHERE permission_type = 'owner';
CREATE UNIQUE INDEX uq_DateiPermission_user ON DateiPermission(datei_id, user_account_id) WHERE user_account_id IS NOT NULL;
CREATE UNIQUE INDEX uq_DateiPermission_group ON DateiPermission(datei_id, user_group_id) WHERE user_group_id IS NOT NULL;

-- ============================================================================
-- Public Link
-- ============================================================================

CREATE TABLE PublicLink (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  token TEXT NOT NULL UNIQUE,
  created_by UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  permission_type TEXT NOT NULL DEFAULT 'read_only',
  expires_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  CONSTRAINT ck_PublicLink_permission_type CHECK (
    permission_type IN ('read_only', 'read_write')
  )
);

CREATE INDEX idx_PublicLink_created_by ON PublicLink(created_by);
CREATE INDEX idx_PublicLink_expires_at ON PublicLink(expires_at) WHERE expires_at IS NOT NULL;

-- ============================================================================
-- Public Link Datei (Relation Table)
-- ============================================================================

CREATE TABLE PublicLink_Datei (
  public_link_id UUID NOT NULL REFERENCES PublicLink(id) ON DELETE CASCADE,
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  PRIMARY KEY (public_link_id, datei_id)
);

CREATE INDEX idx_PublicLink_Datei_datei_id ON PublicLink_Datei(datei_id);

-- ============================================================================
-- Datei Star (User-scoped favorites, Relation Table)
-- ============================================================================

CREATE TABLE Datei_Star (
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  PRIMARY KEY (user_account_id, datei_id)
);

CREATE INDEX idx_Datei_Star_datei_id ON Datei_Star(datei_id);

-- ============================================================================
-- Datei Comment
-- ============================================================================

CREATE TABLE DateiComment (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  datei_id UUID NOT NULL REFERENCES Datei(id) ON DELETE CASCADE,
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp,
  updated_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_DateiComment_datei_id ON DateiComment(datei_id);
CREATE INDEX idx_DateiComment_user_account_id ON DateiComment(user_account_id);

-- ============================================================================
-- Audit Log
-- ============================================================================

CREATE TABLE AuditLog (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  actor_id UUID REFERENCES UserAccount(id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  target_type TEXT NOT NULL,
  target_id UUID NOT NULL,
  metadata JSONB,
  ip_address TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE INDEX idx_AuditLog_actor_id ON AuditLog(actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX idx_AuditLog_target ON AuditLog(target_type, target_id);
CREATE INDEX idx_AuditLog_action ON AuditLog(action);
CREATE INDEX idx_AuditLog_created_at ON AuditLog(created_at);
