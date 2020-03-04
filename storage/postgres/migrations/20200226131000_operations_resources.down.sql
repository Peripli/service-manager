BEGIN;

ALTER TABLE operations DROP COLUMN transitive_resources;

COMMIT;