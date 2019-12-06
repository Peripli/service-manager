BEGIN;

ALTER TABLE operations DROP CONSTRAINT operations_external_id_key;

COMMIT;