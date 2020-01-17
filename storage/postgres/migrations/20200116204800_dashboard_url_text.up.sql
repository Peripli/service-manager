BEGIN;

ALTER TABLE service_instances ALTER COLUMN dashboard_url TYPE text;

COMMIT;