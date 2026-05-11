ALTER TABLE link_projection RENAME COLUMN access_token TO key;
ALTER TABLE link_projection RENAME CONSTRAINT link_projection_access_token_key TO link_projection_key_key;
ALTER TABLE link_projection RENAME CONSTRAINT link_projection_access_token_not_null TO link_projection_key_not_null;

ALTER TABLE link_projection ADD COLUMN open_count BIGINT NOT NULL DEFAULT 0;
