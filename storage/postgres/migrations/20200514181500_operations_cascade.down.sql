BEGIN;

ALTER TABLE operations DROP COLUMN cascade_root_id;
ALTER TABLE operations DROP COLUMN parent_id;

COMMIT;