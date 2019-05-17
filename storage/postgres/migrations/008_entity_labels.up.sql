BEGIN;

CREATE TABLE platform_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255) NOT NULL CHECK (val <> ''),
  platform_id varchar(100) NOT NULL REFERENCES platforms (id) ON DELETE CASCADE,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, platform_id)
);

CREATE TABLE service_offering_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255) NOT NULL CHECK (val <> ''),
  service_offering_id varchar(100) NOT NULL REFERENCES service_offerings (id) ON DELETE CASCADE,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, service_offering_id)
);

CREATE TABLE service_plan_labels
(
  id            varchar(100) PRIMARY KEY,
  key           varchar(255) NOT NULL CHECK (key <> ''),
  val           varchar(255) NOT NULL CHECK (val <> ''),
  service_plan_id varchar(100) NOT NULL REFERENCES service_plans (id) ON DELETE CASCADE,
  created_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, service_plan_id)
);

COMMIT;