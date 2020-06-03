BEGIN;

ALTER TABLE operations ADD COLUMN IF NOT EXISTS cascade_root_id varchar(100);
ALTER TABLE operations ADD COLUMN IF NOT EXISTS parent_id varchar(100);

COMMIT;

