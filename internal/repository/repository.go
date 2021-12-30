// Package repository stores positions in the database and in cache
package repository

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
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

// SignUp func creates new user
func (r *Repository) SignUp(ctx context.Context, deposit float32) (*model.User, error) {
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

// OpenPosition func opens position. Returns id of position, error
func (r *Repository) OpenPosition(ctx context.Context, position *request.OpenPositionRepository, t time.Time) (int32, error) {
	rows, err := r.conn.Query(ctx, "INSERT INTO positions (id, user_id, symbol_id, symbol_title, count, price_open, " +
		"time_open, price_close, time_close, stop_loss, take_profit, is_buy) " +
		"VALUES (nextval('positions_sequence'), $1, $2, $3, $4, $5, $6, NULL, NULL, $7, $8, $9) RETURNING id;",
		position.UserID, position.SymbolID, position.SymbolTitle, position.Count, position.PriceOpen, t, position.StopLoss, position.TakeProfit, position.IsBuy)
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

// ClosePosition func closes position
func (r *Repository) ClosePosition(ctx context.Context, position *request.ClosePosition) error {
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

func (r *Repository) GetPosition(ctx context.Context, positionID int32) (*model.Position, error) {
	var position model.Position
	err := r.conn.QueryRow(ctx, "SELECT id, user_id, symbol_id, symbol_title, count, price_open, time_open, " +
		"stop_loss, take_profit, is_buy FROM positions WHERE id = $1", positionID).Scan(&position.ID, &position.UserID,
			&position.SymbolID, &position.SymbolTitle, &position.Count, &position.PriceOpen, &position.TimeOpen,
			&position.StopLoss, &position.TakeProfit, &position.IsBuy)
	if err != nil {
		return nil, err
	}
	return &position, nil
}

// GetOpenPositions returns all open positions for the certain user
func (r *Repository) GetOpenPositions(userID int32) (map[int32]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var count int32
	r.conn.QueryRow(ctx, "SELECT count(*) FROM positions WHERE user_id = $1 AND price_close is NULL", userID).Scan(&count)

	rows, err := r.conn.Query(ctx, "SELECT id, user_id, symbol_id, symbol_title, count, price_open, time_open, stop_loss, take_profit, is_buy " +
		"FROM positions WHERE user_id = $1 AND price_close is NULL", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make(map[int32]*model.Position)
	for rows.Next() {
		position := model.Position{}
		err = rows.Scan(&position.ID, &position.UserID, &position.SymbolID, &position.SymbolTitle, &position.Count,
			&position.PriceOpen, &position.TimeOpen, &position.StopLoss, &position.TakeProfit, &position.IsBuy)
		if err != nil {
			return nil, err
		}
		positions[position.ID] = &position
	}
	return positions, nil
}

// GetAllOpenPositions returns all open positions
func (r *Repository) GetAllOpenPositions() (map[int32]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := r.conn.Query(ctx, "SELECT id, user_id, symbol_id, symbol_title, count, price_open, time_open, stop_loss, take_profit, is_buy " +
		"FROM positions WHERE price_close is NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positions := make(map[int32]*model.Position)
	for rows.Next() {
		var position model.Position
		err = rows.Scan(&position.ID, &position.UserID, &position.SymbolID, &position.SymbolTitle, &position.Count,
			&position.PriceOpen, &position.TimeOpen, &position.StopLoss, &position.TakeProfit, &position.IsBuy)
		if err != nil {
			return nil, err
		}
		positions[position.ID] = &position
	}
	return positions, nil
}

func (r *Repository) GetAllUsers() (map[int32]*model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := r.conn.Query(ctx, "SELECT id, balance FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make(map[int32]*model.User)
	for rows.Next() {
		var user model.User
		err = rows.Scan(&user.ID, &user.Balance)
		if err != nil {
			return nil, err
		}
		users[user.ID] = &user
	}
	return users, nil
}

// ChangeBalance changes user's balance
func (r *Repository) ChangeBalance(ctx context.Context, userID int32, sum float32) error {
	commandTag, err := r.conn.Exec(ctx, "UPDATE users SET balance = balance + $1 where id = $2", sum, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errors.New("balance didn't change")
	}
	return nil
}
