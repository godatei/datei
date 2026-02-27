-- Projection Tables: Read Models for Event Store
-- These tables are denormalized views updated by event handlers

-- ============================================================================
-- Datei Projection (Current state of datei entities)
-- ============================================================================

CREATE TABLE datei_projection (
  id UUID PRIMARY KEY,
  parent_id UUID REFERENCES datei(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_datei_id UUID REFERENCES datei(id) ON DELETE SET NULL,
  latest_name_id UUID REFERENCES datei_name(id) ON DELETE RESTRICT,
  latest_version_id UUID REFERENCES datei_version(id) ON DELETE RESTRICT,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  trashed_at TIMESTAMPTZ,
  trashed_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  projection_version INT NOT NULL DEFAULT 1 -- Which event version created this state
);

-- Same indexes as original datei table
CREATE INDEX idx_datei_projection_parent_id ON datei_projection(parent_id);
CREATE INDEX idx_datei_projection_linked_datei_id ON datei_projection(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_datei_projection_trashed_at ON datei_projection(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_datei_projection_created_by ON datei_projection(created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_datei_projection_latest_name_id ON datei_projection(latest_name_id) WHERE latest_name_id IS NOT NULL;
CREATE INDEX idx_datei_projection_latest_version_id ON datei_projection(latest_version_id) WHERE latest_version_id IS NOT NULL;

-- ============================================================================
-- Datei Name Projection (Name history)
-- ============================================================================

CREATE TABLE datei_name_projection (
  id UUID PRIMARY KEY,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_datei_name_projection_datei_id ON datei_name_projection(datei_id);

-- ============================================================================
-- Datei Version Projection (Version history with full-text search)
-- ============================================================================

CREATE TABLE datei_version_projection (
  id UUID PRIMARY KEY,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  s3_key TEXT NOT NULL,
  file_size BIGINT NOT NULL,
  checksum TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  content_md TEXT,
  content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content_md, ''))) STORED,
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_datei_version_projection_datei_id ON datei_version_projection(datei_id);
CREATE INDEX idx_datei_version_projection_content_search ON datei_version_projection USING GIN(content_search);

-- ============================================================================
-- Datei Permission Projection (Access control)
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
