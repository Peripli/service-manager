BEGIN;

ALTER TABLE operations ADD COLUMN platform_id varchar(100) NOT NULL DEFAULT '';

COMMIT;