BEGIN;

CREATE TABLE platforms (
    id varchar(100) PRIMARY KEY NOT NULL,
    type varchar(255) NOT NULL,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    created_at timestamp DEFAULT current_timestamp NOT NULL,
    updated_at timestamp DEFAULT current_timestamp NOT NULL,
    username varchar(255) NOT NULL UNIQUE,
    password varchar(500) NOT NULL
);

CREATE TABLE brokers (
    id varchar(100) PRIMARY KEY NOT NULL,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    broker_url text NOT NULL,
    created_at timestamp DEFAULT current_timestamp NOT NULL,
    updated_at timestamp DEFAULT current_timestamp NOT NULL,
    username varchar(255) NOT NULL,
    password varchar(500) NOT NULL,
    catalog json NOT NULL
);

COMMIT;