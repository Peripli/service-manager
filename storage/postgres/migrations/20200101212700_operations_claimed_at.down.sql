BEGIN;

ALTER TABLE operations DROP COLUMN claimed_at;

COMMIT;
