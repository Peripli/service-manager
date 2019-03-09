BEGIN;

ALTER TABLE brokers DROP CONSTRAINT unique_broker_url;

COMMIT;