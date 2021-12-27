// Package request has structs
package request

// Position needed to transmit information about opening or closing a position
type Position struct {
	Act        string
	PositionID int32
	StockID    int32
	Count      int32
	Price      float32
	StopLoss   float32
	TakeProfit float32
}

// ClosePosition is struct
type ClosePosition struct {
	ID         int32
	PriceClose float32
}

// GetAllPositions is struct for getting all open positions from database
type GetAllPositions struct {
	StockID    int32
	PositionID int32
	Count      int32
	PriceOpen  float32
	StopLoss   float32
	TakeProfit float32
}

type BoxInfo struct {
	UserID  int32
	GrpcID  string
	Deposit float32
}
