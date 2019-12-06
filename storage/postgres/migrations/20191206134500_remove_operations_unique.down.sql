BEGIN;

ALTER TABLE operations ADD CONSTRAINT operations_external_id_key UNIQUE (external_id);

COMMIT;