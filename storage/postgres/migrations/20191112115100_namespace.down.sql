BEGIN;

ALTER TABLE notifications DROP COLUMN namespace;
ALTER TABLE brokers DROP COLUMN namespace;
ALTER TABLE platforms DROP COLUMN namespace;
ALTER TABLE service_plans DROP COLUMN namespace;
ALTER TABLE service_offerings DROP COLUMN namespace;
ALTER TABLE visibilities DROP COLUMN namespace;

COMMIT;
