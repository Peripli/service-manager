BEGIN;

ALTER TABLE notifications ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE brokers ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE platforms ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE service_plans ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE service_offerings ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';
ALTER TABLE visibilities ADD COLUMN namespace varchar(64) NOT NULL DEFAULT '';

COMMIT;
