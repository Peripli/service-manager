BEGIN;
ALTER TABLE brokers DROP COLUMN sm_provided_credentials;
COMMIT;