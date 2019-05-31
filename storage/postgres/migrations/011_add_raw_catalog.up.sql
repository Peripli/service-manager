BEGIN;

ALTER TABLE brokers ADD COLUMN catalog json;

END;