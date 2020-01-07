BEGIN;

ALTER TABLE service_offerings DROP COLUMN allow_context_updates;
ALTER TABLE service_plans DROP COLUMN maximum_polling_duration;
ALTER TABLE service_plans DROP COLUMN maintenance_info;

COMMIT;