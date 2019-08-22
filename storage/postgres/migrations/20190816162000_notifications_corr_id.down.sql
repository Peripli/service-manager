BEGIN;

ALTER TABLE notifications DROP COLUMN correlation_id;

COMMIT;