BEGIN;

CREATE TYPE operation_type AS ENUM ('CREATE', 'DELETE', 'UPDATE');
CREATE TYPE operation_state AS ENUM ('SUCCEEDED', 'FAILED', 'IN PROGRESS');

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
  errors            json         DEFAULT '{}'
);

COMMIT;
