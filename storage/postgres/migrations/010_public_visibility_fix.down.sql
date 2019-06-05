BEGIN;

-- There is no way to use the old unique_public_plan_visibility
ALTER TABLE visibilities DROP CONSTRAINT unique_public_plan_visibility;

COMMIT;