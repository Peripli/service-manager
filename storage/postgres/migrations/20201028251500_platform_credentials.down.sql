BEGIN;

ALTER TABLE platforms DROP COLUMN IF EXISTS old_username;
ALTER TABLE platforms DROP COLUMN IF EXISTS old_password;
ALTER TABLE platforms DROP COLUMN IF EXISTS credentials_active;

COMMIT;