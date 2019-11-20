BEGIN;

ALTER TABLE notifications DROP COLUMN paging_sequence;

COMMIT;