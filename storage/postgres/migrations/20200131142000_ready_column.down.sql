BEGIN;

ALTER TABLE brokers DROP COLUMN ready;
ALTER TABLE notifications DROP COLUMN ready;
ALTER TABLE operations DROP COLUMN ready;
ALTER TABLE platforms DROP COLUMN ready;
ALTER TABLE service_offerings DROP COLUMN ready;
ALTER TABLE service_plans DROP COLUMN ready;
ALTER TABLE visibilities DROP COLUMN ready;

COMMIT;