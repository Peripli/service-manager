BEGIN;

ALTER TABLE service_instances DROP COLUMN IF EXISTS shareable;

COMMIT;