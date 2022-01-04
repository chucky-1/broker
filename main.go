package main

import (
	"github.com/caarlos0/env/v6"
	"github.com/chucky-1/broker/internal/config"
	"github.com/chucky-1/broker/internal/grpc/server"
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/service"
	"github.com/chucky-1/broker/protocol"
	pricer "github.com/chucky-1/pricer/protocol"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"context"
	"fmt"
	"net"
	"strconv"
)

const countOfSymbols = 5

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

	// Initial dependencies
	symbols := map[int32]*model.Symbol{}
	symbolID := make([]int32, 0, countOfSymbols)
	for i := 0; i < countOfSymbols; i++ {
		title := fmt.Sprint("Symbol ", strconv.Itoa(i+1))
		symbols[int32(i+1)] = &model.Symbol{
			ID:    int32(i+1),
			Title: title,
		}
		symbolID = append(symbolID, int32(i+1))
	}
	ch := make(chan *model.Price) // this chan is listened in main.go
	chSrv := make(chan *model.Price) // this chan is listened in service.go
	ctx := context.Background()
	rep := repository.NewRepository(conn)
	srv, err := service.NewService(ctx, rep, chSrv, symbols)
	if err != nil {
		log.Fatal(err)
	}

	// Grpc Broker
	go func() {
		hostAndPort := fmt.Sprint(cfg.HostGrpcServer, ":", cfg.PortGrpcServer)
		lis, err := net.Listen("tcp", hostAndPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		protocol.RegisterBrokerServer(s, server.NewServer(srv))
		log.Infof("server listening at %v", lis.Addr())
		if err = s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Grpc Prices
	go func() {
		hostAndPort := fmt.Sprint(cfg.HostGrpcClient, ":", cfg.PortGrpcClient)
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
		client := pricer.NewPricesClient(clientConn)
		stream, err := client.Subscribe(context.Background())
		if err != nil {
			return
		}
		err = stream.Send(&pricer.SubscribeRequest{
			Action:   0,
			PriceId: symbolID,
		})
		if err != nil {
			return
		}

		for {
			select {
			case <-stream.Context().Done():
				return
			default:
				price, err := stream.Recv()
				if err != nil {
					log.Error(err)
					continue
				}
				s := &model.Price{
					ID:   price.PriceId,
					Bid:  price.Bid,
					Ask:  price.Ask,
					Time: price.Update.Seconds,
				}
				// ch <- s
				chSrv <- s
			}
		}
	}()

	for {
		symbol := <-ch
		log.Info(symbol)
	}
}
