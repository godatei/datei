-- Event Store: Event Sourcing Infrastructure
-- Stores all domain events for event-driven architecture

-- ============================================================================
-- Event Store Table
-- ============================================================================

CREATE TABLE event_store (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  event_metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_event_stream_version CHECK (stream_version > 0)
);

-- Ensure idempotency: each stream version is unique
CREATE UNIQUE INDEX uq_event_store_stream_version ON event_store(stream_id, stream_version);

-- Fast lookup of events for a specific datei
CREATE INDEX idx_event_store_stream_id ON event_store(stream_id);

-- Timeline queries
CREATE INDEX idx_event_store_created_at ON event_store(created_at DESC);

-- Event type queries for subscriptions
CREATE INDEX idx_event_store_event_type ON event_store(event_type);

-- ============================================================================
-- Event Snapshots Table
-- ============================================================================

CREATE TABLE event_snapshots (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  aggregate_state JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_snapshot_stream_version CHECK (stream_version > 0)
);

-- Latest snapshot per stream for fast aggregate reconstruction
CREATE UNIQUE INDEX uq_event_snapshots_stream_version ON event_snapshots(stream_id, stream_version DESC);

-- Timezone-aware timestamp for retention policies
CREATE INDEX idx_event_snapshots_created_at ON event_snapshots(created_at DESC);
