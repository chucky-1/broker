// Package repository stores positions in the database
package repository

import (
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
	"strconv"

	"context"
	"time"
)

// Repository works with postgres
type Repository struct {
	conn  *pgx.Conn
	cache *Cache
}

// NewRepository is constructor
func NewRepository(conn *pgx.Conn, cache *Cache) *Repository {
	return &Repository{conn: conn, cache: cache}
}

// Open func opens position
func (r *Repository) Open(ctx context.Context, clientID int, stockID string) (int, error) {
	stock, err := r.cache.Get(stockID)
	if err != nil {
		return 0, err
	}
	t, err := getTime(stock.Update)
	if err != nil {
		log.Error(err)
	}

	rows, err := r.conn.Query(ctx, "INSERT INTO swops (id, client_id, stock_id, price_open, time_open, price_close, time_close) " +
		"VALUES (nextval('swops_sequence'), $1, $2, $3, $4, NULL, NULL) RETURNING id",
		clientID, stock.ID, stock.Price, t)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if err = rows.Err(); err != nil {
		return 0, err
	}

	var id int
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

// Close func closes position and returns income or loss
func (r *Repository) Close(ctx context.Context, swopID int, stockID string) (float32, error) {
	stock, err := r.cache.Get(stockID)
	if err != nil {
		return 0, err
	}

	rows, err := r.conn.Query(ctx, "UPDATE swops SET price_close = $1, time_close = $2 WHERE id = $3 RETURNING price_open",
		stock.Price, time.Now(), swopID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if err = rows.Err(); err != nil {
		return 0, err
	}

	var priceOpen float32
	for rows.Next() {
		err = rows.Scan(&priceOpen)
		if err != nil {
			return 0, err
		}
	}

	delta := stock.Price - priceOpen
	return delta, nil
}

// getTime formats a string to a date. String from Redis ID
func getTime(id string) (time.Time, error) {
	mkr, err := strconv.Atoi(id)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Unix(int64(mkr)/1000, 0)
	return t, nil
}
