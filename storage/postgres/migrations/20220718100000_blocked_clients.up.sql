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


CREATE OR REPLACE FUNCTION new_blocked_client() RETURNS TRIGGER AS $$
  DECLARE
data json;

BEGIN
    data = json_build_object(
      'id', NEW.id,
      'client_id', NEW.client_id,
      'subaccount_id', NEW.subaccount_id,
      'blocked_methods', NEW.blocked_methods
    );
    PERFORM pg_notify('new_blocked_client', data::text);

    -- Result is ignored since this is an AFTER trigger
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER value_insert
    AFTER INSERT ON blocked_clients
    FOR EACH ROW EXECUTE PROCEDURE new_blocked_client();

COMMIT;