BEGIN;

ALTER TABLE brokers DROP COLUMN paging_sequence;
ALTER TABLE platforms DROP COLUMN paging_sequence;
ALTER TABLE visibilities DROP COLUMN paging_sequence;
ALTER TABLE service_plans DROP COLUMN paging_sequence;
ALTER TABLE service_offerings DROP COLUMN paging_sequence;

COMMIT;