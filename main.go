package main

import (
	"github.com/caarlos0/env/v6"
	"github.com/chucky-1/broker/internal/config"
	"github.com/jackc/pgx/v4"

	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
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

		}
	}(conn, context.Background())
}
