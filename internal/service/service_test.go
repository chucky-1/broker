package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestService_stopLoss(t *testing.T) {
	testTable := []struct {
		name     string
		position *model.Position
		price    *model.Price
		expect   bool
	}{
		{
			name: "OK if isBuy is true",
			position: &model.Position{
				StopLoss: 1000,
				IsBuy:    true,
			},
			price: &model.Price{
				//Bid: 1100,
				Ask: 990,
			},
			expect: true,
		},
		{
			name: "OK if isBuy is false",
			position: &model.Position{
				StopLoss: 1000,
				IsBuy:    false,
			},
			price: &model.Price{
				Bid: 1100,
				//Ask: 990,
			},
			expect: true,
		},
		{
			name: "Failed if isBuy is true",
			position: &model.Position{
				StopLoss: 1000,
				IsBuy:    true,
			},
			price: &model.Price{
				//Bid: 1100,
				Ask: 1100,
			},
			expect: false,
		},
		{
			name: "Failed if isBuy is false",
			position: &model.Position{
				StopLoss: 1000,
				IsBuy:    false,
			},
			price: &model.Price{
				Bid: 900,
				//Ask: 990,
			},
			expect: false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			b := stopLoss(testCase.position, testCase.price)
			assert.Equal(t, testCase.expect, b)
		})
	}
}

func TestService_takeProfit(t *testing.T) {
	testTable := []struct {
		name     string
		position *model.Position
		price    *model.Price
		expect   bool
	}{
		{
			name: "OK if isBuy is true",
			position: &model.Position{
				TakeProfit: 1000,
				IsBuy:      true,
			},
			price: &model.Price{
				//Bid: 1100,
				Ask: 1100,
			},
			expect: true,
		},
		{
			name: "OK if isBuy is false",
			position: &model.Position{
				TakeProfit: 1000,
				IsBuy:      false,
			},
			price: &model.Price{
				Bid: 900,
				//Ask: 990,
			},
			expect: true,
		},
		{
			name: "Failed if isBuy is true",
			position: &model.Position{
				TakeProfit: 1000,
				IsBuy:      true,
			},
			price: &model.Price{
				//Bid: 1100,
				Ask: 900,
			},
			expect: false,
		},
		{
			name: "Failed if isBuy is false",
			position: &model.Position{
				TakeProfit: 1000,
				IsBuy:      true,
			},
			price: &model.Price{
				Bid: 1100,
				//Ask: 990,
			},
			expect: false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			b := takeProfit(testCase.position, testCase.price)
			assert.Equal(t, testCase.expect, b)
		})
	}
}
