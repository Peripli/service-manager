BEGIN;

ALTER TABLE brokers DROP COLUMN tls_client_key;
ALTER TABLE brokers DROP COLUMN tls_client_certificate;

COMMIT;