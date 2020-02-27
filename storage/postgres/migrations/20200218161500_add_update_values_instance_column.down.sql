BEGIN;

ALTER TABLE service_instances DROP COLUMN update_values;

COMMIT;