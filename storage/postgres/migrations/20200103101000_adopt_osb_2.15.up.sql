BEGIN;

ALTER TABLE service_offerings ADD COLUMN allow_context_updates BOOLEAN NOT NULL DEFAULT '0';
ALTER TABLE service_plans ADD COLUMN maximum_polling_duration INTEGER NOT NULL DEFAULT 0;
ALTER TABLE service_plans ADD COLUMN maintenance_info json NOT NULL DEFAULT '{}';

COMMIT;