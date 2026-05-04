-- Datei: Link Domain (public sharing)

-- ============================================================================
-- Event Store
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
  access_token TEXT NOT NULL UNIQUE,
  -- Plain-text shared secret (intentional; the value is displayed back to the
  -- owner so they can share it with viewers out-of-band).
  code TEXT,
  expires_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_link_projection_owner_id ON link_projection(owner_id);

-- ============================================================================
-- Link <-> Datei Join Projection
-- ============================================================================

CREATE TABLE link_datei_projection (
  link_id UUID NOT NULL REFERENCES link_projection(id) ON DELETE CASCADE,
  datei_id UUID NOT NULL REFERENCES datei_projection(id) ON DELETE CASCADE,
  added_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (link_id, datei_id)
);

CREATE INDEX idx_link_datei_projection_datei_id ON link_datei_projection(datei_id);
