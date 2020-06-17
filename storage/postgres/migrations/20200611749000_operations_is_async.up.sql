BEGIN;

ALTER TABLE operations ADD COLUMN IF NOT EXISTS is_async boolean;

COMMIT;

