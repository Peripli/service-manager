BEGIN;

CREATE TABLE platforms (
    id varchar(100) PRIMARY KEY,
    type varchar(255) NOT NULL,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    created_at timestamp DEFAULT current_timestamp,
    updated_at timestamp DEFAULT current_timestamp,
    username varchar(255) NOT NULL UNIQUE,
    password varchar(500)
);

CREATE TABLE brokers (
    id varchar(100) PRIMARY KEY,
    name varchar(255) NOT NULL UNIQUE,
    description text,
    broker_url text NOT NULL,
    created_at timestamp DEFAULT current_timestamp,
    updated_at timestamp DEFAULT current_timestamp,
    username varchar(255),
    password varchar(500),
    catalog json
);

COMMIT;