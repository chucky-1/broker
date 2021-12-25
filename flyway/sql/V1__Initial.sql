CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    balance numeric
);

CREATE TABLE positions (
    id SERIAL PRIMARY KEY,
    user_id SERIAL REFERENCES users(id) NOT NULL,
    stock_id integer NOT NULL,
    stock_name varchar(40) NOT NULL,
    count integer NOT NULL,
    price_open numeric NOT NULL,
    time_open timestamp,
    price_close numeric,
    time_close timestamp
);

CREATE SEQUENCE users_sequence;
CREATE SEQUENCE positions_sequence;