// Package request has structs
package request

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

type PositionByUser struct {
	Action     string
	UserID     int32
	PositionID int32
}
