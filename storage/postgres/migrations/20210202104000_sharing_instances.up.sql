ALTER TABLE service_instances   ADD COLUMN IF NOT EXISTS referenced_instance_id     varchar(250);
ALTER TABLE service_instances   ADD COLUMN IF NOT EXISTS shareable                  boolean DEFAULT FALSE;
ALTER TABLE service_plans       ADD COLUMN IF NOT EXISTS shareable                  boolean DEFAULT FALSE;


