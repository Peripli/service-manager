BEGIN;

ALTER TABLE service_instances DROP COLUMN dashboard_url;

COMMIT;