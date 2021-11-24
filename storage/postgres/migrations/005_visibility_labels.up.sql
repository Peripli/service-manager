BEGIN;

CREATE TABLE visibility_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255),
  visibility_id varchar(100) NOT NULL REFERENCES visibilities (id) ON DELETE CASCADE,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, visibility_id)
);

COMMIT;