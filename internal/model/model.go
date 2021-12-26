// Package model has struct of essence
package model

// User is model of user
type User struct {
	ID      int32
	Balance float32
}

// Position is model of position
type Position struct {
	ID         int32
	UserID     int32
	StockID    int32
	StockTitle string
	Count      int32
	PriceOpen  float32
}
