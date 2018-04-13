BEGIN;

CREATE SEQUENCE "credentials_id_seq"
    INCREMENT 1
    START 1
    MINVALUE 1
    MAXVALUE 2147483647
    CACHE 1;

CREATE TABLE IF NOT EXISTS credentials (
    id integer PRIMARY KEY NOT NULL DEFAULT nextval('"credentials_id_seq"'::regclass),
    type integer NOT NULL,
    username varchar(255),
    password varchar(500),
    token text
);

CREATE TABLE IF NOT EXISTS platforms (
    id varchar(100) PRIMARY KEY,
    type varchar(255) NOT NULL,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    created_at timestamp DEFAULT current_timestamp,
    updated_at timestamp DEFAULT current_timestamp,
    credentials_id integer NOT NULL,
    FOREIGN KEY (credentials_id) REFERENCES credentials
);

CREATE TABLE IF NOT EXISTS brokers (
    id varchar(100) PRIMARY KEY,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    broker_url text NOT NULL,
    created_at timestamp DEFAULT current_timestamp,
    updated_at timestamp DEFAULT current_timestamp,
    credentials_id integer NOT NULL,
    FOREIGN KEY (credentials_id) REFERENCES credentials
);

COMMIT;