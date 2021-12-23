package main

import (
	"github.com/caarlos0/env/v6"
	"github.com/chucky-1/broker/internal/config"
	"github.com/chucky-1/broker/internal/grpc/server"
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/protocol"
	"github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"context"
	"fmt"
	"net"
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

	chn := make(chan *protocol.Stock) // this chan is listened in server.go

	// Grpc Swops
	var srv *server.Server
	go func() {
		hostAndPort = fmt.Sprint(cfg.HostGrpcServer, ":", cfg.PortGrpcServer)
		lis, err := net.Listen("tcp", hostAndPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		srv = server.NewServer(rep, chn)
		protocol.RegisterSwopsServer(s, srv)
		log.Infof("server listening at %v", lis.Addr())
		if err = s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	ch := make(chan *protocol.Stock) // this chan is listened in main.go

	// Grpc Prices
	go func() {
		hostAndPort = fmt.Sprint(cfg.HostGrpcClient, ":", cfg.PortGrpcClient)
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
			case <-stream.Context().Done():
				return
			default:
				st, err := stream.Recv()
				if err != nil {
					log.Error(err)
					continue
				}
				if srv.GetNum() > 0 {
					chn <- st
				}
				err = cch.Set(&model.Stock{
					ID:     st.Id,
					Title:  st.Title,
					Price:  st.Price,
					Update: st.Update,
				})
				if err != nil {
					log.Error(err)
				} else {
					ch <- st
				}
			}
		}
	}()

	for {
		stock := <-ch
		log.Info(stock)
	}
}
