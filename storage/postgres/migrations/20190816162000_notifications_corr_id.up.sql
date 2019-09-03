BEGIN;

ALTER TABLE notifications ADD COLUMN correlation_id varchar(40);

COMMIT;