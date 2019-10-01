BEGIN;

ALTER TABLE brokers ADD COLUMN paging_sequence BIGSERIAL;
ALTER TABLE platforms ADD COLUMN paging_sequence BIGSERIAL;
ALTER TABLE visibilities ADD COLUMN paging_sequence BIGSERIAL;
ALTER TABLE service_plans ADD COLUMN paging_sequence BIGSERIAL;
ALTER TABLE service_offerings ADD COLUMN paging_sequence BIGSERIAL;

COMMIT;