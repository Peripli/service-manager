BEGIN;

ALTER TABLE operations ADD COLUMN IF NOT EXISTS transitive_resources json;

COMMIT;