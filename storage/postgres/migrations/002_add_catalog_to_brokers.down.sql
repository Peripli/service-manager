BEGIN;

ALTER TABLE brokers DROP COLUMN IF EXISTS catalog;

END;