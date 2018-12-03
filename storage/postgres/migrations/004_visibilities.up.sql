BEGIN;

CREATE TABLE service_visibilities (
   id varchar(100) PRIMARY KEY,
   platform_id varchar(255) REFERENCES platforms(id) ON DELETE CASCADE,
   service_plan_id varchar(255) NOT NULL REFERENCES service_plans(id) ON DELETE CASCADE,

   UNIQUE (platform_id, service_plan_id)
);

CREATE OR REPLACE FUNCTION check_unique_public_plan(spid varchar, pid varchar)
   RETURNS boolean AS
   $$
      DECLARE
      i int;
      BEGIN
         SELECT COUNT(*) INTO i FROM service_visibilities WHERE service_plan_id = spid AND platform_id IS NULL;
         IF (i > 0) THEN
            RETURN false;
         END IF;

         IF (pid IS NULL) THEN
            SELECT COUNT(*) INTO i FROM service_visibilities WHERE service_plan_id = spid AND platform_id IS NOT NULL;
            IF (i > 0) THEN
               RETURN false;
            END IF;
         END IF;

         RETURN true;
      END
   $$ LANGUAGE plpgsql;
END;

ALTER TABLE service_visibilities ADD CONSTRAINT unique_public_visibility CHECK (check_unique_public_plan(service_plan_id, platform_id));


