BEGIN;

ALTER TABLE service_instances ADD COLUMN update_values json DEFAULT '{}';

COMMIT;