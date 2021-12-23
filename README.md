# broker

Depends on pricer https://github.com/chucky-1/pricer

The broker receives prices from the pricer by GRPC stream. Prices are stored in the cache.

The client can buy and sell stocks. This means opening and closing positions. All positions are stored in the database. 
Each client has a balance that is stored in the database.

The broker automatically updates the asset value and informs the client if the stock price that the client owns has changed.

The client https://github.com/chucky-1/trader