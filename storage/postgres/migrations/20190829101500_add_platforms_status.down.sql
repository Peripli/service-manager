BEGIN;

ALTER TABLE platforms DROP COLUMN active;
ALTER TABLE platforms DROP COLUMN last_active;

COMMIT;