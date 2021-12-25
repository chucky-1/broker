// Package repository stores positions in the database and in cache
package repository

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/jackc/pgx/v4"

	"context"
	"errors"
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

// CreateUser func creates new user
func (r *Repository) CreateUser(deposit float32) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var id int32
	err := r.conn.QueryRow(ctx, "INSERT INTO users (id, balance) VALUES (nextval('users_sequence'), $1) RETURNING id",
		deposit).Scan(&id)
	if err != nil {
		return nil, err
	}
	return &model.User{ID: id, Balance: deposit}, nil
}

// SignIn gets user from database
func (r *Repository) SignIn(id int32) (*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var user model.User
	err := r.conn.QueryRow(ctx, "SELECT id, balance FROM users WHERE id = $1", id).Scan(&user.ID, &user.Balance)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Open func opens position. Returns id of position, error
func (r *Repository) Open(position *model.Position, t time.Time) (int32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := r.conn.Query(ctx, "INSERT INTO positions (id, user_id, stock_id, stock_name, count, price_open, " +
		"time_open, price_close, time_close) VALUES (nextval('positions_sequence'), $1, $2, $3, $4, $5, $6, NULL, NULL) " +
		"RETURNING id;", position.UserID, position.StockID, position.StockTitle, position.Count, position.PriceOpen, t)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int32
	for rows.Next() {
		rows.Scan(&id)
	}
	rows.Close()
	if rows.CommandTag().RowsAffected() == 0 {
		return 0, errors.New("position didn't open")
	}
	return id, nil
}

// Close func closes position
func (r *Repository) Close(position *model.ClosePosition) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "UPDATE positions SET price_close = $1, time_close = CURRENT_TIMESTAMP " +
		"WHERE id = $2", position.PriceClose, position.ID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("position didn't close")
	}
	return nil
}

// GetOpenPositions returns all open positions
func (r *Repository) GetOpenPositions(userID int32) (map[int32]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var count int32
	r.conn.QueryRow(ctx, "SELECT count(*) FROM positions WHERE user_id = $1 AND price_close is NULL", userID).Scan(&count)

	rows, err := r.conn.Query(ctx, "SELECT id, user_id, stock_id, stock_name, count, price_open " +
		"FROM positions WHERE user_id = $1 AND price_close is NULL", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make(map[int32]*model.Position)
	for rows.Next() {
		position := model.Position{}
		err = rows.Scan(&position.ID, &position.UserID, &position.StockID,
			&position.StockTitle, &position.Count, &position.PriceOpen)
		if err != nil {
			return nil, err
		}
		positions[position.ID] = &position
	}
	return positions, nil
}

// ChangeBalance changes user's balance
func (r *Repository) ChangeBalance(userID int32, sum float32) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	commandTag, err := r.conn.Exec(ctx, "UPDATE users SET balance = balance + $1 where id = $2", sum, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("balance didn't change")
	}
	return nil
}
