BEGIN;

ALTER TABLE service_instances RENAME COLUMN previous_values TO new_state;

COMMIT;