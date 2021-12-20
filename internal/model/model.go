package model

// Client is struct of client
type Client struct {
	ID      int
	balance float32
}

// Stock contains fields that describe the shares of companies
type Stock struct {
	ID     int32   `validate:"required"`
	Title  string  `validate:"required"`
	Price  float32 `validate:"required,gt=0"`
	Update string  `validate:"required"`
}
