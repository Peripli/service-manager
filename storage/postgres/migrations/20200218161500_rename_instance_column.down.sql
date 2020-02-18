BEGIN;

ALTER TABLE service_instances RENAME COLUMN new_state TO previous_values;

COMMIT;