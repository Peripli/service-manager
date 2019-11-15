BEGIN;

ALTER TABLE notifications ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE brokers ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE platforms ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE service_plans ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE service_offerings ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE visibilities ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';

ALTER TABLE brokers DROP CONSTRAINT brokers_name_key;
-- TODO transfer subaccount labels into namespace
ALTER TABLE brokers ADD CONSTRAINT unique_broker UNIQUE (name, namespace);

ALTER TABLE platforms DROP CONSTRAINT platforms_name_key;
-- TODO transfer subaccount labels into namespace
ALTER TABLE platforms ADD CONSTRAINT unique_platform UNIQUE (name, namespace);

COMMIT;
