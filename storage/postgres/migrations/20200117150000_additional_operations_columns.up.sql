BEGIN;

ALTER TABLE operations ADD COLUMN reschedule BOOLEAN NOT NULL DEFAULT '0';
ALTER TABLE operations ADD COLUMN deletion_scheduled TIMESTAMP NOT NULL DEFAULT '0001-01-01 00:00:00+00';
ALTER TABLE operations ADD COLUMN external BOOLEAN NOT NULL DEFAULT '0';

COMMIT;