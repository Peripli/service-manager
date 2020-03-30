BEGIN;

ALTER TABLE brokers ADD COLUMN tls_client_key bytea NOT NULL DEFAULT '';
ALTER TABLE brokers ADD COLUMN tls_client_certificate text NOT NULL DEFAULT '';

COMMIT;