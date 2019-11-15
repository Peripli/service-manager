BEGIN;

ALTER TABLE notifications DROP COLUMN namespace;
ALTER TABLE brokers DROP COLUMN namespace;
ALTER TABLE platforms DROP COLUMN namespace;
ALTER TABLE service_plans DROP COLUMN namespace;
ALTER TABLE service_offerings DROP COLUMN namespace;
ALTER TABLE visibilities DROP COLUMN namespace;

ALTER TABLE brokers DROP CONSTRAINT unique_broker;
-- TODO transfer namespace to subaccount labels and fix duplicate names
ALTER TABLE brokers ADD CONSTRAINT brokers_name_key UNIQUE (name);

ALTER TABLE platforms DROP CONSTRAINT unique_platform;
-- TODO transfer subaccount labels into namespace
ALTER TABLE platforms ADD CONSTRAINT platforms_name_key UNIQUE (name);

COMMIT;
