BEGIN;

ALTER TABLE operations DROP COLUMN virtual;
ALTER TABLE operations DROP COLUMN cascade;
ALTER TABLE operations DROP COLUMN parent;

COMMIT;