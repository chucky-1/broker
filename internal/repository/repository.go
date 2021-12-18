// Package repository stores positions in the database
package repository

import (
	"github.com/jackc/pgx/v4"

	"context"
	"time"
)

// Repository works with postgres
type Repository struct {
	conn *pgx.Conn
}

// NewRepository is constructor
func NewRepository(conn *pgx.Conn) *Repository {
	return &Repository{conn: conn}
}

// Open func opens position
func (r *Repository) Open(ctx context.Context, clientID int, stockID int32, stockPrice int32, timeOpen time.Time) (int, error) {
	rows, err := r.conn.Query(ctx, "INSERT INTO swops (id, client_id, stock_id, price_open, time_open, price_close, time_close) " +
		"VALUES (nextval('swops_sequence'), $1, $2, $3, $4, NULL, NULL) RETURNING id",
		clientID, stockID, stockPrice, timeOpen)
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
func (r *Repository) Close(ctx context.Context, priceClose float32, timeClose time.Time, swopID int) (float32, error) {
	rows, err := r.conn.Query(ctx, "UPDATE swops SET price_close = $1, time_close = $2 WHERE id = $3 RETURNING price_open",
		priceClose, timeClose, swopID)
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

	delta := priceClose - priceOpen
	return delta, nil
}
