BEGIN;

CREATE TABLE broker_platform_credentials
(
  id                varchar(100) PRIMARY KEY,

  username          varchar(500) NOT NULL,
  password_hash     varchar(500) NOT NULL,
  old_username      varchar(500) NOT NULL,
  old_password_hash varchar(500) NOT NULL,

  platform_id       varchar(100) NOT NULL REFERENCES platforms (id) ON DELETE CASCADE,
  broker_id         varchar(100) NOT NULL REFERENCES brokers (id) ON DELETE CASCADE,

  created_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paging_sequence   BIGSERIAL,

  ready             boolean NOT NULL,

  UNIQUE (platform_id, broker_id)
);

CREATE TABLE broker_platform_credential_labels
(
  id                             varchar(100) PRIMARY KEY,
  key                            varchar(255) NOT NULL CHECK (key <> ''),
  val                            varchar(255) NOT NULL CHECK (val <> ''),
  broker_platform_credential_id  varchar(100) NOT NULL REFERENCES broker_platform_credentials (id) ON DELETE CASCADE,
  created_at                     timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at                     timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (key, val, broker_platform_credential_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS broker_platform_credentials_paging_sequence_uindex
  on broker_platform_credentials (paging_sequence);


COMMIT;