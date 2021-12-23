// Package repository stores positions in the database and in cache
package repository

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"

	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const startBalance = 1000

// Repository works with postgres
type Repository struct {
	conn  *pgx.Conn
	cache *Cache
}

// NewRepository is constructor
func NewRepository(conn *pgx.Conn, cache *Cache) *Repository {
	return &Repository{conn: conn, cache: cache}
}

// CreateUser func creates new user
func (r *Repository) CreateUser(grpcID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "INSERT INTO users (grpc_id, balance) VALUES ($1, $2)", grpcID, startBalance)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("user didn't create")
	}
	return nil
}

// Open func opens position. Returns id of position, error
func (r *Repository) Open(grpcID string, stockID, count int32) (int, error) {
	stcID := strconv.Itoa(int(stockID))
	stock, err := r.cache.Get(stcID)
	if err != nil {
		return 0, fmt.Errorf("stock with id %d didn't find", stockID)
	}

	t, err := getTime(stock.Update)
	if err != nil {
		log.Error(err)
	}

	currentBalance, err := r.GetBalance(grpcID)
	if err != nil {
		return 0, err
	}
	sum := stock.Price * float32(count)
	ok := r.checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}
	err = r.changeBalance(grpcID, -sum)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := r.conn.Query(ctx, "INSERT INTO positions (id, grpc_id, stock_id, price_open, count, time_open, price_close, time_close)" +
		"VALUES (nextval('positions_sequence'), $1, $2, $3, $4, $5, NULL, NULL) RETURNING id;",
		grpcID, stock.ID, stock.Price, count, t)
	if err != nil {
		r.returnBalance(grpcID, currentBalance)
		return 0, err
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		rows.Scan(&id)
	}

	rows.Close()
	if rows.CommandTag().RowsAffected() == 0 {
		r.returnBalance(grpcID, currentBalance)
		return 0, errors.New("position didn't open")
	}
	return id, nil
}

// Close func closes position and returns income or loss
func (r *Repository) Close(grpcID string, positionID int32) error {
	stockID, count, err := r.GetInfo(positionID)
	if err != nil {
		return err
	}

	stcID := strconv.Itoa(int(stockID))
	stock, err := r.cache.Get(stcID)
	if err != nil {
		return fmt.Errorf("stock with id %d didn't find", stockID)
	}

	currentBalance, err := r.GetBalance(grpcID)
	if err != nil {
		return err
	}
	sum := stock.Price * float32(count)
	err = r.changeBalance(grpcID, sum)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "UPDATE positions SET price_close = $1, time_close = CURRENT_TIMESTAMP " +
		"WHERE id = $2", stock.Price, positionID)
	if err != nil {
		r.returnBalance(grpcID, currentBalance)
		return err
	}
	if commandTag.RowsAffected() != 1 {
		r.returnBalance(grpcID, currentBalance)
		return errors.New("position didn't close")
	}
	return nil
}

// GetBalance returns user's balance
func (r *Repository) GetBalance(grpcID string) (float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var balance float32
	err := r.conn.QueryRow(ctx, "SELECT balance FROM users WHERE grpc_id = $1", grpcID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

// GetInfo returns id of stock and count of stock
func (r *Repository) GetInfo(positionID int32) (int32, int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var stockID, count int32
	err := r.conn.QueryRow(ctx, "SELECT stock_id, count FROM positions WHERE id = $1", positionID).Scan(&stockID, &count)
	if err != nil {
		return 0, 0, err
	}
	return stockID, count, nil
}

// GetOpenPositions returns all open positions
func (r *Repository) GetOpenPositions(grpcID string) ([]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var count int32
	r.conn.QueryRow(ctx, "SELECT count(*) FROM positions WHERE grpc_id = $1 AND price_close is NULL").Scan(&count)

	rows, err := r.conn.Query(ctx, "SELECT id, stock_id, count FROM positions WHERE grpc_id = $1 AND price_close is NULL", grpcID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var position model.Position
	positions := make([]*model.Position, count)
	for rows.Next() {
		err = rows.Scan(&position.ID, &position.StockID, &position.Count)
		if err != nil {
			return nil, err
		}
		positions = append(positions, &position)
	}
	return positions, nil
}

// GetBalanceRealTime returns the total value of the stocks, slice with detailed information and error
func (r *Repository) GetBalanceRealTime(positions []*model.Position) (float32, []*model.Detail, error) {
	details := make([]*model.Detail, 0, len(positions))
	var sum float32
	for _, position := range positions {
		stock, err := r.cache.Get(strconv.Itoa(int(position.StockID)))
		if err != nil {
			return 0, nil, err
		}
		detail := model.Detail{
			Stock: stock,
			Count: position.Count,
		}
		details = append(details, &detail)
		sum += stock.Price * float32(position.Count)
	}
	return sum, details, nil
}

func (r *Repository) changeBalance(grpcID string, sum float32) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "UPDATE users SET balance = balance + $1 where grpc_id = $2", sum, grpcID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("balance didn't change")
	}
	return nil
}

// Return true if enough money and false if not enough money
func (r *Repository) checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}

func (r *Repository) returnBalance(grpcID string, balance float32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "UPDATE users SET balance = $1 where grpc_id = $2", balance, grpcID)
	if err != nil {
		log.Error("balance is not restored")
	}
	if commandTag.RowsAffected() != 1 {
		log.Error("balance is not restored")
	}
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
