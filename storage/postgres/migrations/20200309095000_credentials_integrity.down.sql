BEGIN;

ALTER TABLE brokers DROP COLUMN IF EXISTS integrity;
ALTER TABLE platforms DROP COLUMN IF EXISTS integrity;
ALTER TABLE service_bindings DROP COLUMN IF EXISTS integrity;
ALTER TABLE broker_platform_credentials DROP COLUMN IF EXISTS integrity;

COMMIT;