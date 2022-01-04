// Package request has structs
package request

import (
	"context"
	"github.com/chucky-1/broker/internal/model"
)

// OpenPositionRepository stores parameters for opening a position in the repository
type OpenPositionRepository struct {
	UserID      int32
	SymbolID    int32
	SymbolTitle string
	Count       int32
	PriceOpen   float32
	StopLoss    float32
	TakeProfit  float32
	IsBuy       bool
}

// OpenPositionService stores parameters for opening a position in the service
type OpenPositionService struct {
	UserID     int32
	SymbolID   int32
	Price      float32
	Count      int32
	StopLoss   float32
	TakeProfit float32
	IsBuy      bool
}

// ClosePosition stores fields when closing a position
type ClosePosition struct {
	ID         int32
	PriceClose float32
}

type PositionCloser interface {
	Close(ctx context.Context, position *model.Position) error
}
