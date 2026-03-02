-- Projection Tables: Read Models for Event Store
-- These tables are denormalized views updated by event handlers
-- Single datei_projection table containing current state with embedded latest name/version data

-- ============================================================================
-- Datei Projection (Current state of datei entities)
-- ============================================================================

CREATE TABLE datei_projection (
  id UUID PRIMARY KEY,
  parent_id UUID REFERENCES datei(id) ON DELETE RESTRICT,
  is_directory BOOLEAN NOT NULL DEFAULT false,
  linked_datei_id UUID REFERENCES datei(id) ON DELETE SET NULL,
  -- Latest name data (embedded, no separate history table needed - history is in event store)
  latest_name TEXT NOT NULL,
  -- Latest version data (embedded, no separate history table needed - history is in event store)
  latest_version_s3_key TEXT,
  latest_version_file_size BIGINT,
  latest_version_checksum TEXT,
  latest_version_mime_type TEXT,
  latest_version_content_md TEXT,
  latest_version_content_search TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', coalesce(latest_version_content_md, ''))) STORED,
  -- Core metadata
  created_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  trashed_at TIMESTAMPTZ,
  trashed_by UUID REFERENCES user_account(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  projection_version INT NOT NULL DEFAULT 1 -- Which event version created this state
);

-- Indexes for common queries
CREATE INDEX idx_datei_projection_parent_id ON datei_projection(parent_id);
CREATE INDEX idx_datei_projection_linked_datei_id ON datei_projection(linked_datei_id) WHERE linked_datei_id IS NOT NULL;
CREATE INDEX idx_datei_projection_trashed_at ON datei_projection(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_datei_projection_created_by ON datei_projection(created_by) WHERE created_by IS NOT NULL;
CREATE INDEX idx_datei_projection_latest_version_content_search ON datei_projection USING GIN(latest_version_content_search);

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
