BEGIN;

ALTER TABLE operations ADD COLUMN reschedule_timestamp timestamptz NOT NULL DEFAULT '0001-01-01 00:00:00+00';

COMMIT;