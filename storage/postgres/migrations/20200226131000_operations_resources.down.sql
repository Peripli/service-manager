BEGIN;

ALTER TABLE operations ALTER COLUMN transitive_resources DROP;

COMMIT;