BEGIN;

ALTER TABLE brokers DROP COLUMN IF EXISTS catalog;

ALTER TABLE brokers ADD CONSTRAINT unique_broker_url UNIQUE (broker_url);

CREATE TABLE service_offerings (
    id                    varchar(100) PRIMARY KEY NOT NULL,
    name                  varchar(255) UNIQUE      NOT NULL,
    description           text                     NOT NULL DEFAULT '',
    created_at            timestamp                NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            timestamp                NOT NULL DEFAULT CURRENT_TIMESTAMP,
    catalog_id            varchar(255)             NOT NULL,
    catalog_name          varchar(255)             NOT NULL,

    bindable              boolean  NOT NULL DEFAULT '0',
    plan_updateable        boolean  NOT NULL DEFAULT '0',
    instances_retrievable boolean  NOT NULL DEFAULT '0',
    bindings_retrievable  boolean  NOT NULL DEFAULT '0',

    metadata              json                     NOT NULL DEFAULT '{}',
    tags                  json                     NOT NULL DEFAULT '{}',
    requires              json                     NOT NULL DEFAULT '{}',

    broker_id     varchar(100)             NOT NULL REFERENCES brokers(id) ON DELETE CASCADE,
    UNIQUE (catalog_id, broker_id)
);

CREATE TABLE service_plans (
   id varchar(100) PRIMARY KEY NOT NULL,
   name varchar(255) UNIQUE NOT NULL,
   description text NOT NULL DEFAULT '',
   created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
   catalog_name varchar(255) NOT NULL,
   catalog_id varchar(255) NOT NULL,

   bindable boolean NOT NULL,
   plan_updateable boolean NOT NULL,
   free boolean NOT NULL,

   metadata json NOT NULL DEFAULT '{}',
   create_instance_schema json NOT NULL DEFAULT '{}',
   update_instance_schema json NOT NULL DEFAULT '{}',
   create_binding_schema json NOT NULL DEFAULT '{}',

   service_offering_id varchar(100) NOT NULL REFERENCES service_offerings(id) ON DELETE CASCADE,
   UNIQUE (service_offering_id, name)
);

COMMIT;