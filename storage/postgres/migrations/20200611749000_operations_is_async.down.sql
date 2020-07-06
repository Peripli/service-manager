BEGIN;

ALTER TABLE operations DROP COLUMN is_async;

COMMIT;

