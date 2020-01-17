BEGIN;

ALTER TABLE service_instances ALTER COLUMN dashboard_url TYPE varchar(16000);

COMMIT;