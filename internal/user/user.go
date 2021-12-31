// Package user handles each user
package user

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/internal/service"
	log "github.com/sirupsen/logrus"

	"context"
	"errors"
	"time"
)

// User keeps state each user
type User struct {
	srv         *service.Service
	id          int32
	balance     float32
	chPrice     chan *model.Price
	positions   map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
}

// NewUser is constructor
func NewUser(ctx context.Context, srv *service.Service, id int32, balance float32) (*User, error) {
	u := User{
		srv:         srv,
		id:          id,
		balance:     balance,
		chPrice:     make(chan *model.Price),
		positions:   make(map[int32]map[int32]*model.Position),
	}

	rep, mu := srv.GetRepository()
	mu.Lock()
	positions, err := rep.GetOpenPositions(u.id)
	mu.Unlock()
	if err != nil {
		return nil, err
	}
	for _, position := range positions {
		allPositions, ok := u.positions[position.SymbolID]
		if !ok {
			u.positions[position.SymbolID] = make(map[int32]*model.Position)
			u.positions[position.SymbolID][position.ID] = position
		} else {
			allPositions[position.ID] = position
		}
	}

	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- u.chPrice:
				for _, position := range u.positions[price.ID] {
					p := pnl(position, price)
					log.Infof("pnl for position %d is %f", position.ID, p)
					if stopLoss(position, price) {
						u.close(ctx, position.ID)
					}
					if takeProfit(position, price) {
						u.close(ctx, position.ID)
					}
				}
			}
		}
	}(ctx)
	return &u, nil
}

// OpenPosition opens position of user
func (u *User) OpenPosition(ctx context.Context, r *request.OpenPositionService) (int32, error) {
	var price float32
	prices, muPrices := u.srv.GetPrices()
	if r.IsBuy == true {
		muPrices.RLock()
		price = prices[r.SymbolID].Bid
		muPrices.RUnlock()
		ok := checkPrice(price, r.Price, r.IsBuy)
		if !ok {
			return 0, errors.New("price changed. Try again")
		}
	} else {
		muPrices.RLock()
		price = prices[r.SymbolID].Ask
		muPrices.RUnlock()
		ok := checkPrice(price, r.Price, r.IsBuy)
		if !ok {
			return 0, errors.New("price changed. Try again")
		}
	}
	currentBalance := u.balance
	sum := price * float32(r.Count)
	ok := u.checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}

	rep, muRep :=u.srv.GetRepository()
	muRep.Lock()
	err := rep.ChangeBalance(ctx, r.UserID, -sum)
	muRep.Unlock()
	if err != nil {
		return 0, err
	}
	u.balance -= sum

	t := time.Now()

	symbols, muSym := u.srv.GetSymbols()
	muSym.RLock()
	title := symbols[r.SymbolID].Title
	muSym.RUnlock()

	muRep.Lock()
	id, err := rep.OpenPosition(ctx, &request.OpenPositionRepository{
		UserID:      r.UserID,
		SymbolID:    r.SymbolID,
		SymbolTitle: title,
		Count:       r.Count,
		PriceOpen:   price,
		StopLoss:    r.StopLoss,
		TakeProfit:  r.TakeProfit,
		IsBuy:       r.IsBuy,
	}, t)
	muRep.Unlock()
	if err != nil {
		u.balance += sum
		muRep.Lock()
		err = rep.ChangeBalance(ctx, u.id, sum)
		muRep.Unlock()
		if err != nil {
			log.Error(err)
		}
		return 0, err
	}

	position := model.Position {
		ID: id,
		UserID: r.UserID,
		SymbolID: r.SymbolID,
		SymbolTitle: "",
		Count: r.Count,
		PriceOpen: price,
		TimeOpen: t,
		StopLoss: r.StopLoss,
		TakeProfit: r.TakeProfit,
		IsBuy: r.IsBuy,
	}
	allPositions, ok := u.positions[position.SymbolID]
	if !ok {
		u.positions[position.SymbolID] = make(map[int32]*model.Position)
		u.positions[position.SymbolID][position.ID] = &position
	} else {
		allPositions[position.ID] = &position
	}
	return id, nil
}

// ClosePosition closes position of user
func (u *User) ClosePosition(ctx context.Context, positionID int32) error {
	rep, muRep := u.srv.GetRepository()
	muRep.Lock()
	position, err := rep.GetPosition(ctx, positionID)
	muRep.Unlock()

	if err != nil {
		return err
	}
	var price float32
	prices, muPrices := u.srv.GetPrices()
	if position.IsBuy == true {
		muPrices.RLock()
		price = prices[position.SymbolID].Ask
		muPrices.RUnlock()
	} else {
		muPrices.RLock()
		price = prices[position.SymbolID].Bid
		muPrices.RUnlock()
	}
	closePosition := request.ClosePosition{
		ID:         positionID,
		PriceClose: price,
	}
	sum := price * float32(position.Count)
	muRep.Lock()
	err = rep.ChangeBalance(ctx, u.id, sum)
	muRep.Unlock()
	if err != nil {
		return err
	}
	u.balance += sum
	muRep.Lock()
	err = rep.ClosePosition(ctx, &closePosition)
	muRep.Unlock()
	if err != nil {
		u.balance -= sum
		muRep.Lock()
		err = rep.ChangeBalance(ctx, u.id, -sum)
		muRep.Unlock()
		if err != nil {
			log.Error(err)
		}
		return err
	}
	delete(u.positions[position.SymbolID], positionID)
	return nil
}

// SetBalance changes balance of user
func (u *User) SetBalance(ctx context.Context, sum float32) error {
	rep, muRep := u.srv.GetRepository()
	muRep.Lock()
	err := rep.ChangeBalance(ctx, u.id, sum)
	muRep.Unlock()
	if err != nil {
		return err
	}
	u.balance += sum
	return nil
}

// GetBalance returns balance
func (u *User) GetBalance() float32 {
	return u.balance
}

// Return true if enough money and false if not enough money
func (u *User) checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}

func (u *User) close(ctx context.Context, positionID int32) {
	err := u.ClosePosition(ctx, positionID)
	if err != nil {
		log.Error(err)
	}
}

// pnl is Profit and loss. Shows how much you earned or lost
func pnl(position *model.Position, price *model.Price) float32 {
	return price.Bid * float32(position.Count) - position.PriceOpen * float32(position.Count)
}

func stopLoss(position *model.Position, price *model.Price) bool {
	if position.IsBuy {
		return price.Ask <= position.StopLoss
	}
	return price.Bid >= position.StopLoss
}

func takeProfit(position *model.Position, price *model.Price) bool {
	if position.IsBuy {
		return price.Ask >= position.TakeProfit
	}
	return price.Bid <= position.TakeProfit
}

func checkPrice(priceActual, priceWait float32, isBuy bool) bool {
	if isBuy {
		return priceWait >= priceActual
	}
	return priceWait <= priceActual
}

// GetID returns id
func (u *User) GetID() int32 {
	return u.id
}

// GetChanPrice returns chan of price
func (u *User) GetChanPrice() chan *model.Price {
	return u.chPrice
}

// GetPositions returns positions
func (u *User) GetPositions() map[int32]map[int32]*model.Position {
	return u.positions
}
