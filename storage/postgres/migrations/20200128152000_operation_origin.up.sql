BEGIN;

ALTER TABLE operations ADD COLUMN origin varchar(18) NOT NULL DEFAULT 'external';

COMMIT;