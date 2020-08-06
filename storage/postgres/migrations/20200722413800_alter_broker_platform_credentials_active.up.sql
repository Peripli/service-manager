BEGIN;

ALTER TABLE broker_platform_credentials ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT false;

COMMIT;