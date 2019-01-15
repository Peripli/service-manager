BEGIN;

CREATE TABLE broker_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255) NOT NULL CHECK (val <> ''),
  broker_id varchar(100) NOT NULL REFERENCES brokers (id) ON DELETE CASCADE,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, broker_id)
);

COMMIT;