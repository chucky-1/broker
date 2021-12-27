package main

import (
	"context"
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/chucky-1/broker/internal/config"
	"github.com/chucky-1/broker/internal/grpc/server"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/internal/service"
	"github.com/chucky-1/broker/protocol"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
	rep := repository.NewRepository(conn)

	// Initializing dependencies
	chSrvAdd := make(chan *service.Service)
	chSrvDel := make(chan string)
	chPosition := make(chan *request.Position)
	chStock := make(chan *protocol.Stock)
	err = service.NewBot(rep, chSrvAdd, chSrvDel, chPosition, chStock)
	if err != nil {
		log.Error(err)
	}
	srvStore := make(map[string]*service.Service) // map[grpcID]*srv
	chanBoxInfo := make(chan *request.BoxInfo)
	chanSrvDel := make(chan string) // For grpcID
	chanService := make(chan *service.Service)
	go func() {
		go func() {
			for grpcID := range chanSrvDel {
				delete(srvStore, grpcID)
			}
		}()
		for {
			info := <-chanBoxInfo
			srv, err := service.NewService(rep, info.UserID, info.GrpcID, info.Deposit, chPosition)
			if err != nil {
				log.Error(err)
			} else {
				chanService <- srv
				srvStore[info.GrpcID] = srv
				chSrvAdd <- srv
			}
		}
	}()

	// Grpc Positions
	go func() {
		hostAndPort := fmt.Sprint(cfg.HostGrpcServer, ":", cfg.PortGrpcServer)
		lis, err := net.Listen("tcp", hostAndPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		protocol.RegisterPositionsServer(s, server.NewServer(chanBoxInfo, chanService, chanSrvDel, chSrvDel))
		log.Infof("server listening at %v", lis.Addr())
		if err = s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	ch := make(chan *protocol.Stock) // this chan is listened in main.go

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
				stock, err := stream.Recv()
				if err != nil {
					log.Error(err)
					continue
				}
				if len(srvStore) > 0 {
					go func() {
						for _, s := range srvStore {
							s.GetChan() <- stock
						}
					}()
				}
				chStock <- stock
				//ch <- stock
			}
		}
	}()

	for {
		stock := <-ch
		log.Info(stock)
	}
}
