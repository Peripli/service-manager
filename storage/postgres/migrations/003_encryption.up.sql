BEGIN;

CREATE SCHEMA vault;

CREATE TABLE vault.safe (
  secret bytea NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

ALTER TABLE credentials
    ALTER COLUMN password TYPE bytea USING(password::bytea);

CREATE ROLE access_user;

GRANT USAGE ON SCHEMA vault TO access_user;
GRANT SELECT, INSERT ON vault.safe TO access_user;

-- This requires guard to be a user in the database
GRANT access_user TO guard;

REVOKE ALL ON SCHEMA public FROM access_user;

COMMIT;