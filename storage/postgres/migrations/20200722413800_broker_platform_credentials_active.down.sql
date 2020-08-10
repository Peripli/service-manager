BEGIN;

ALTER TABLE broker_platform_credentials DROP COLUMN active;

COMMIT;