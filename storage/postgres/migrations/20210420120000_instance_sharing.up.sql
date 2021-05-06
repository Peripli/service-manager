ALTER TABLE service_instances ADD COLUMN IF NOT EXISTS referenced_instance_id varchar(100);
ALTER TABLE service_instances ADD COLUMN IF NOT EXISTS shared boolean;