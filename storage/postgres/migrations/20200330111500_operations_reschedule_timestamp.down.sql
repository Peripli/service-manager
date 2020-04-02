BEGIN;

ALTER TABLE operations DROP COLUMN reschedule_timestamp;

COMMIT;