CREATE TABLE clients (
    id SERIAL PRIMARY KEY,
    balance numeric
);

CREATE TABLE stocks (
    id SERIAL PRIMARY KEY,
    title varchar(40) NOT NULL,
    price numeric CHECK (price > 0)
);

CREATE TABLE swops (
    id SERIAL PRIMARY KEY,
    client_id SERIAL REFERENCES clients(id) NOT NULL,
    stock_id SERIAL REFERENCES stocks(id) NOT NULL,
    price_open numeric NOT NULL,
    time_open timestamp,
    price_close numeric,
    time_close timestamp
);

CREATE SEQUENCE clients_sequence;
CREATE SEQUENCE stocks_sequence;
CREATE SEQUENCE swops_sequence;