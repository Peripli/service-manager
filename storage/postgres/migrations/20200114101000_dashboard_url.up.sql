BEGIN;

ALTER TABLE service_instances ADD COLUMN dashboard_url varchar(16000);

COMMIT;