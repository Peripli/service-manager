BEGIN;

ALTER TABLE service_instances   DROP COLUMN IF EXISTS referenced_instance_id;
ALTER TABLE service_instances   DROP COLUMN IF EXISTS shareable;
ALTER TABLE service_plans       DROP COLUMN IF EXISTS shareable;

COMMIT;
