CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    balance numeric
);

CREATE TABLE positions (
    id SERIAL PRIMARY KEY,
    user_id SERIAL REFERENCES users(id) NOT NULL,
    symbol_id integer NOT NULL,
    symbol_title varchar(40) NOT NULL,
    count integer NOT NULL,
    price_open numeric NOT NULL,
    time_open timestamp,
    price_close numeric,
    time_close timestamp,
    stop_loss numeric,
    take_profit numeric,
    is_buy boolean
);

CREATE SEQUENCE users_sequence;
CREATE SEQUENCE positions_sequence;