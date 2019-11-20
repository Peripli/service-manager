BEGIN;

CREATE TABLE service_instances
(
  id                varchar(100) PRIMARY KEY,
  name              varchar(100) NOT NULL,
  service_plan_id   varchar(100) NOT NULL REFERENCES service_plans (id),
  platform_id       varchar(100) NOT NULL REFERENCES platforms (id),
  maintenance_info  json DEFAULT '{}',
  context           json DEFAULT '{}',
  previous_values   json DEFAULT '{}',
  created_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  usable            boolean NOT NULL,
  ready             boolean NOT NULL,
  paging_sequence   BIGSERIAL
);

CREATE TABLE service_instance_labels
(
  id                  varchar(100) PRIMARY KEY,
  key                 varchar(255) NOT NULL CHECK (key <> ''),
  val                 varchar(255) NOT NULL CHECK (val <> ''),
  service_instance_id varchar(100) NOT NULL REFERENCES service_instances (id) ON DELETE CASCADE,
  created_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, service_instance_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS service_instances_paging_sequence_uindex
  on service_instances (paging_sequence);

COMMIT;