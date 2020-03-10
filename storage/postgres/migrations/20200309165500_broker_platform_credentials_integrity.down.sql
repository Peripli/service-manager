BEGIN;

ALTER TABLE broker_platform_credentials DROP COLUMN IF EXISTS password_hash;
ALTER TABLE broker_platform_credentials ADD COLUMN IF NOT EXISTS password_hash varchar(500);

ALTER TABLE broker_platform_credentials DROP COLUMN IF EXISTS old_password_hash;
ALTER TABLE broker_platform_credentials ADD COLUMN IF NOT EXISTS old_password_hash varchar(500);

COMMIT;