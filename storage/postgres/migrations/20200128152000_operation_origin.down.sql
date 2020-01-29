BEGIN;

ALTER TABLE operations DROP COLUMN platform_id;

COMMIT;