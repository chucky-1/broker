// Package model has struct of essence
package model

import "time"

// Stock contains fields that describe the shares of companies
type Stock struct {
	ID     int32   `validate:"required"`
	Title  string  `validate:"required"`
	Price  float32 `validate:"required,gt=0"`
	Update string  `validate:"required"`
}

// Swop is model of position
type Swop struct {
	ID         int32
	GrpcID     string
	StockID    int32
	PriceOpen  float32
	Count      int32
	TimeOpen   time.Time
	PriceClose float32
	TimeClose  time.Time
}

// Detail contains detailed information about swop
type Detail struct {
	Stock *Stock
	Count int32
}
