BEGIN;

ALTER TABLE brokers ADD CONSTRAINT unique_broker_url UNIQUE (broker_url);

COMMIT;