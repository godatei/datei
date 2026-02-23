CREATE TABLE UserAccount (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  password_salt TEXT NOT NULL,
  mfa_secret TEXT,
  mfa_enabled BOOLEAN NOT NULL DEFAULT false,
  mfa_enabled_at TIMESTAMPTZ,
  archived_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
  CONSTRAINT ck_UserAccount_mfa CHECK (
    mfa_enabled = false OR mfa_secret IS NOT NULL
  )
);
