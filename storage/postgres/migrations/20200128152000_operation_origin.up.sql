BEGIN;

ALTER TABLE operations ADD COLUMN origin varchar(18) NOT NULL DEFAULT 'external';
UPDATE operations SET origin = 'internal' WHERE resource_type != '/v1/service_instances';

COMMIT;