ALTER TABLE users
    ADD COLUMN auth_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE users
    ADD CONSTRAINT users_auth_version_positive CHECK (auth_version > 0);
