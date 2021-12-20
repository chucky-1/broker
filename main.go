package main

import (
	"context"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/chucky-1/broker/internal/config"
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/protocol"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	// Configuration
	cfg := new(config.Config)
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("%v", err)
	}

	// Postgres
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.UsernamePostgres, cfg.PasswordPostgres, cfg.HostPostgres, cfg.PortPostgres, cfg.DBNamePostgres)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer func(conn *pgx.Conn, ctx context.Context) {
		err = conn.Close(ctx)
		if err != nil {
			log.Error(err)
		}
	}(conn, context.Background())

	// Redis cache
	hostAndPort := fmt.Sprint(cfg.HostRedisCache, ":", cfg.PortRedisCache)
	ring := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{cfg.ServerRedisCache: hostAndPort}})
	c := cache.New(&cache.Options{Redis: ring})

	cch := repository.NewCache(c)
	rep := repository.NewRepository(conn, cch)
	fmt.Println(rep)

	// Grpc
	go func() {
		hostAndPort = fmt.Sprint(cfg.HostGrpc, ":", cfg.PortGrpc)
		clientConn, err := grpc.Dial(hostAndPort, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("fail to dial: %v", err)
		}
		defer func(conn *grpc.ClientConn) {
			err = conn.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(clientConn)
		client := protocol.NewPricesClient(clientConn)
		stream, err := client.SubAll(context.Background(), &protocol.Request{})
		if err != nil {
			return
		}
		for {
			select {
			case <- stream.Context().Done():
				return
			default:
				st, err := stream.Recv()
				if err != nil {
					log.Error(err)
				}
				err = cch.Set(&model.Stock{
					ID:     st.Id,
					Title:  st.Title,
					Price:  st.Price,
					Update: st.Update,
				})
				if err != nil {
					log.Error(err)
				}
			}
		}
	}()

	chn := make(chan string)
	<- chn
}
