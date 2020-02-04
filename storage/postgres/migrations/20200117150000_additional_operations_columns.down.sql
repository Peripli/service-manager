BEGIN;

ALTER TABLE operations DROP COLUMN reschedule;
ALTER TABLE operations DROP COLUMN deletion_scheduled;

COMMIT;