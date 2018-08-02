BEGIN;

CREATE TABLE safe (
  secret bytea NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

ALTER TABLE platforms
    ALTER COLUMN password TYPE bytea USING(password::bytea);

ALTER TABLE brokers
  ALTER COLUMN password TYPE bytea USING(password::bytea);

COMMIT;