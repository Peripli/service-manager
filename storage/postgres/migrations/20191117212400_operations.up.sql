BEGIN;

CREATE TYPE operation_type  AS ENUM ('create', 'delete', 'update');
CREATE TYPE operation_state AS ENUM ('succeeded', 'failed', 'in progress');

CREATE TABLE operations
(
  id                varchar(100) PRIMARY KEY,
  correlation_id    varchar(100),
  description       varchar(255),
  external_id       varchar(100) UNIQUE,
  type              operation_type NOT NULL,
  state             operation_state NOT NULL,
  resource_type     varchar(100) NOT NULL,
  resource_id       varchar(100),
  created_at        timestamptz    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamptz    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  errors            json         DEFAULT '{}',
  paging_sequence   BIGSERIAL
);

CREATE TABLE operation_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255),
  operation_id  varchar(100) NOT NULL REFERENCES operations (id) ON DELETE CASCADE,
  created_at    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, operation_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS operations_paging_sequence_uindex
    on operations (paging_sequence);

COMMIT;
