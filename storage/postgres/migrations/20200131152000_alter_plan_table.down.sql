BEGIN;

ALTER TABLE service_plans ALTER COLUMN plan_updateable SET NOT NULL;
ALTER TABLE service_plans ALTER COLUMN bindable SET NOT NULL;

COMMIT;