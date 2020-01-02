BEGIN;

ALTER TABLE operations ADD COLUMN claimed_at timestamptz;

COMMIT;
