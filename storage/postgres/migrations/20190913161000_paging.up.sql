BEGIN;

ALTER TABLE brokers ADD COLUMN paging_sequence SERIAL;
ALTER TABLE platforms ADD COLUMN paging_sequence SERIAL;
ALTER TABLE visibilities ADD COLUMN paging_sequence SERIAL;
ALTER TABLE service_plans ADD COLUMN paging_sequence SERIAL;
ALTER TABLE service_offerings ADD COLUMN paging_sequence SERIAL;

COMMIT;