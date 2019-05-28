BEGIN;

ALTER TABLE visibilities DROP CONSTRAINT unique_public_plan_visibility;

CREATE OR REPLACE FUNCTION check_unique_public_plan(visid varchar, spid varchar, pid varchar)
    RETURNS boolean AS
$$
DECLARE
    i int;
BEGIN
    SELECT COUNT(*) INTO i FROM visibilities WHERE service_plan_id = spid AND platform_id IS NULL AND id <> visid;
    IF (i > 0) THEN
        RETURN false;
    END IF;

    IF (pid IS NULL) THEN
        SELECT COUNT(*) INTO i FROM visibilities WHERE service_plan_id = spid AND platform_id IS NOT NULL;
        IF (i > 0) THEN
            RETURN false;
        END IF;
    END IF;

    RETURN true;
END
$$ LANGUAGE plpgsql;
END;

ALTER TABLE visibilities ADD CONSTRAINT unique_public_plan_visibility CHECK (check_unique_public_plan(id, service_plan_id, platform_id));

COMMIT;