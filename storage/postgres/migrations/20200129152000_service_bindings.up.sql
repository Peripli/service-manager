BEGIN;

CREATE TABLE service_bindings
(
  id                  varchar(100) PRIMARY KEY,
  name                varchar(100) NOT NULL,
  service_instance_id varchar(100) NOT NULL REFERENCES service_instances (id),

  syslog_drain_url  text,
  route_service_url text,
  volume_mounts     json,
  endpoints         json,

  credentials   bytea,
  context       json DEFAULT '{}',
  bind_resource json DEFAULT '{}',

  created_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paging_sequence   BIGSERIAL,

  ready             boolean NOT NULL
);

CREATE TABLE service_binding_labels
(
  id                  varchar(100) PRIMARY KEY,
  key                 varchar(255) NOT NULL CHECK (key <> ''),
  val                 varchar(255) NOT NULL CHECK (val <> ''),
  service_binding_id  varchar(100) NOT NULL REFERENCES service_bindings (id) ON DELETE CASCADE,
  created_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, service_binding_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS service_bindings_paging_sequence_uindex
  on service_bindings (paging_sequence);

COMMIT;