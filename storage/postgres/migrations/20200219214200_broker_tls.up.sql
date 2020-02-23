BEGIN;

ALTER TABLE brokers ADD COLUMN tls_client_key bytea;
ALTER TABLE brokers ADD COLUMN tls_client_certificate  text;

COMMIT;