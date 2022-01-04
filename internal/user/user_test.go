package user

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUser_stopLoss(t *testing.T) {
	testTable := []struct {
		name     string
		position *model.Position
		expect   bool
	}{
		{
			name: "OK if isBuy is true",
			position: &model.Position{
				StopLoss: 1000,
				AskClose: 990,
				IsBuy:    true,
			},
			expect: true,
		},
		{
			name: "OK if isBuy is false",
			position: &model.Position{
				StopLoss: 1000,
				BidClose: 1100,
				IsBuy:    false,
			},
			expect: true,
		},
		{
			name: "Failed if isBuy is true",
			position: &model.Position{
				StopLoss: 1000,
				AskClose: 1100,
				IsBuy:    true,
			},
			expect: false,
		},
		{
			name: "Failed if isBuy is false",
			position: &model.Position{
				StopLoss: 1000,
				BidClose: 900,
				IsBuy:    false,
			},
			expect: false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			b := stopLoss(testCase.position)
			assert.Equal(t, testCase.expect, b)
		})
	}
}

func TestUser_takeProfit(t *testing.T) {
	testTable := []struct {
		name     string
		position *model.Position
		expect   bool
	}{
		{
			name: "OK if isBuy is true",
			position: &model.Position{
				TakeProfit: 1000,
				AskClose:   1100,
				IsBuy:      true,
			},
			expect: true,
		},
		{
			name: "OK if isBuy is false",
			position: &model.Position{
				TakeProfit: 1000,
				BidClose:   900,
				IsBuy:      false,
			},
			expect: true,
		},
		{
			name: "Failed if isBuy is true",
			position: &model.Position{
				TakeProfit: 1000,
				AskClose:   900,
				IsBuy:      true,
			},
			expect: false,
		},
		{
			name: "Failed if isBuy is false",
			position: &model.Position{
				TakeProfit: 1000,
				BidClose:   1100,
				IsBuy:      true,
			},
			expect: false,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			b := takeProfit(testCase.position)
			assert.Equal(t, testCase.expect, b)
		})
	}
}
