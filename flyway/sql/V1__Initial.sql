CREATE TABLE users (
    grpc_id varchar(36) PRIMARY KEY,
    balance numeric
);

CREATE TABLE swops (
    id SERIAL PRIMARY KEY,
    grpc_id varchar(36) REFERENCES users(grpc_id) NOT NULL,
    stock_id integer NOT NULL,
    price_open numeric NOT NULL,
    count integer NOT NULL,
    time_open timestamp,
    price_close numeric,
    time_close timestamp
);

CREATE SEQUENCE swops_sequence;