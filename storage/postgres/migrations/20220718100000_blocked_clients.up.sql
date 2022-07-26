BEGIN;

CREATE TABLE blocked_clients
(
  id            varchar(100) PRIMARY KEY,
  client_id     varchar(100) NOT NULL,
  subaccount_id varchar(100),
  blocked_methods       TEXT [],

  created_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paging_sequence   BIGSERIAL,
  ready             boolean NOT NULL
);

CREATE TABLE blocked_clients_labels
(
    id              varchar(100) PRIMARY KEY,
    key             varchar(255) NOT NULL CHECK (key <> ''),
    val             varchar(255) NOT NULL CHECK (val <> ''),
    blocked_client_id    varchar(100) NOT NULL REFERENCES blocked_clients (id) ON DELETE CASCADE,
    created_at      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (key, val, blocked_client_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS blocked_clients_paging_sequence_uindex
  on blocked_clients (paging_sequence);

COMMIT;