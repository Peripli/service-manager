BEGIN;

ALTER TABLE notifications ADD COLUMN paging_sequence BIGSERIAL;

COMMIT;