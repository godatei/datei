ALTER TABLE link_projection DROP COLUMN open_count;

ALTER TABLE link_projection RENAME CONSTRAINT link_projection_key_not_null TO link_projection_access_token_not_null;
ALTER TABLE link_projection RENAME CONSTRAINT link_projection_key_key TO link_projection_access_token_key;
ALTER TABLE link_projection RENAME COLUMN key TO access_token;
