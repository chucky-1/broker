// Package model has struct of essence
package model

import "time"

type Symbol struct {
	ID    int32
	Title string
}

// Price contains fields that describe the shares of companies
type Price struct {
	ID   int32
	Bid  float32
	Ask  float32
	Time int64
}

// User is model of user
type User struct {
	ID      int32
	Balance float32
}

// Position is model of position
type Position struct {
	ID         int32
	UserID     int32
	SymbolID    int32
	SymbolTitle string
	Count      int32
	PriceOpen  float32
	TimeOpen   time.Time
	StopLoss   float32
	TakeProfit float32
	IsBuy      bool
}
