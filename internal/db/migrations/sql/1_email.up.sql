-- File: Email receiving (mail accounts, rules, IMAP polling)

-- ============================================================================
-- Enum Types
-- ============================================================================

CREATE TYPE mail_security AS ENUM ('ssl', 'starttls', 'none');
CREATE TYPE mail_action AS ENUM ('mark_as_read');

-- ============================================================================
-- Event Stores
-- ============================================================================

CREATE TABLE mail_account_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_mail_account_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_mail_account_event_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_mail_account_event_stream_id ON mail_account_event(stream_id);
CREATE INDEX idx_mail_account_event_created_at ON mail_account_event(created_at DESC);
CREATE INDEX idx_mail_account_event_event_type ON mail_account_event(event_type);

CREATE TABLE mail_rule_event (
  id BIGSERIAL PRIMARY KEY,
  stream_id UUID NOT NULL,
  stream_version INT NOT NULL,
  event_type VARCHAR NOT NULL,
  event_data JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT ck_mail_rule_event_stream_version CHECK (stream_version > 0),
  CONSTRAINT uq_mail_rule_event_stream_version UNIQUE (stream_id, stream_version)
);

CREATE INDEX idx_mail_rule_event_stream_id ON mail_rule_event(stream_id);
CREATE INDEX idx_mail_rule_event_created_at ON mail_rule_event(created_at DESC);
CREATE INDEX idx_mail_rule_event_event_type ON mail_rule_event(event_type);

-- ============================================================================
-- Mail Account Projection
-- ============================================================================

CREATE TABLE mail_account_projection (
  id UUID PRIMARY KEY,
  owner_id UUID NOT NULL REFERENCES user_account_projection(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  imap_host TEXT NOT NULL,
  imap_port INT NOT NULL,
  username TEXT NOT NULL,
  -- Symmetrically encrypted IMAP password (AES-256-GCM). Never returned by the API.
  password_encrypted BYTEA NOT NULL,
  security mail_security NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ck_mail_account_projection_port CHECK (imap_port BETWEEN 1 AND 65535)
);

CREATE INDEX idx_mail_account_projection_owner_id ON mail_account_projection(owner_id);

-- ============================================================================
-- Mail Rule Projection
-- ============================================================================

CREATE TABLE mail_rule_projection (
  id UUID PRIMARY KEY,
  account_id UUID NOT NULL REFERENCES mail_account_projection(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT true,
  folder TEXT NOT NULL DEFAULT 'INBOX',
  filter_from TEXT,
  filter_subject TEXT,
  max_age_days INT NOT NULL,
  -- Comma-separated filename globs (e.g. "*.pdf,*.docx"); empty means all attachments.
  attachment_pattern TEXT,
  action mail_action NOT NULL,
  -- Target directory for ingested attachments; NULL means the owner's root.
  target_directory_id UUID REFERENCES file_projection(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ck_mail_rule_projection_max_age CHECK (max_age_days > 0)
);

CREATE INDEX idx_mail_rule_projection_account_id ON mail_rule_projection(account_id);

-- ============================================================================
-- Processed Message Bookkeeping
--
-- Tracks which IMAP messages have already been handled so the poller never
-- reprocesses a mail, independently of whether the rule action (e.g. mark as
-- read) succeeded. This is worker infrastructure state, not domain state, so it
-- is written directly rather than via event sourcing.
-- ============================================================================

CREATE TABLE mail_processed_message (
  id BIGSERIAL PRIMARY KEY,
  account_id UUID NOT NULL REFERENCES mail_account_projection(id) ON DELETE CASCADE,
  folder TEXT NOT NULL,
  uid_validity BIGINT NOT NULL,
  imap_uid BIGINT NOT NULL,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT uq_mail_processed_message UNIQUE (account_id, folder, uid_validity, imap_uid)
);

CREATE INDEX idx_mail_processed_message_account_id ON mail_processed_message(account_id);
