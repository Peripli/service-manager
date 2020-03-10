BEGIN;

ALTER TABLE brokers ADD COLUMN IF NOT EXISTS integrity bytea;
ALTER TABLE platforms ADD COLUMN IF NOT EXISTS integrity bytea;
ALTER TABLE service_bindings ADD COLUMN IF NOT EXISTS integrity bytea;
ALTER TABLE broker_platform_credentials ADD COLUMN IF NOT EXISTS integrity bytea;


COMMIT;