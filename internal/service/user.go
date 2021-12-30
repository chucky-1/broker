package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"

	"context"
	"errors"
	"fmt"
	"time"
)

type User struct {
	srv       *Service
	id        int32
	balance   float32
	chPrice   chan *model.Price
	prices    map[int32]*model.Price    // map[price.ID]*price
	positions map[int32]*model.Position // map[position.ID]*position
	allPositionsBySymbol map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
	chPosByUser          chan *request.PositionByUser
}

func NewUser(ctx context.Context, srv *Service, id int32, balance float32, chPosByUser chan *request.PositionByUser) (*User, error) {
	u := User{
		srv: srv,
		id: id,
		balance: balance,
		chPrice: make(chan *model.Price),
		prices: make(map[int32]*model.Price),
		positions: make(map[int32]*model.Position),
		allPositionsBySymbol: make(map[int32]map[int32]*model.Position),
		chPosByUser: chPosByUser,
	}

	srv.muRep.Lock()
	positions, err := srv.rep.GetOpenPositions(u.id)
	srv.muRep.Unlock()
	if err != nil {
		return nil, err
	}
	for _, position := range positions {
		u.positions[position.ID] = position
		allPositions, ok := u.allPositionsBySymbol[position.SymbolID]
		if !ok {
			u.allPositionsBySymbol[position.SymbolID] = make(map[int32]*model.Position)
			u.allPositionsBySymbol[position.SymbolID][position.ID] = position
		} else {
			allPositions[position.ID] = position
		}
		chPosByUser <- &request.PositionByUser{
			Action:     "ADD",
			UserID:     u.id,
			PositionID: position.ID,
		}
	}

	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- u.chPrice:
				u.prices[price.ID] = price
				for _, position := range u.allPositionsBySymbol[price.ID] {
					pnl, err := u.pnl(position.ID)
					if err != nil {
						log.Error(err)
						continue
					}
					log.Infof("pnl for position %d is %f", position.ID, pnl)
					if stopLoss(position, u.prices[position.SymbolID]) {
						u.close(position.ID)
					}
					if takeProfit(position, u.prices[position.SymbolID]) {
						u.close(position.ID)
					}
				}
			}
		}
	}(ctx)
	return &u, nil
}

func (u *User) OpenPosition(ctx context.Context, r *request.OpenPositionService) (int32, error) {
	var price float32
	if r.IsBuy == true {
		price = u.prices[r.SymbolID].Bid
	} else {
		price = u.prices[r.SymbolID].Ask
	}
	currentBalance := u.balance
	sum := price * float32(r.Count)
	ok := u.checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}
	u.srv.muRep.Lock()
	err := u.srv.rep.ChangeBalance(ctx, r.UserID, -sum)
	u.srv.muRep.Unlock()
	if err != nil {
		return 0, err
	}
	u.balance -= sum

	t := time.Now()

	u.srv.muSymbols.RLock()
	title := u.srv.symbols[r.SymbolID].Title
	u.srv.muSymbols.RUnlock()

	u.srv.muRep.Lock()
	id, err := u.srv.rep.OpenPosition(ctx, &request.OpenPositionRepository{
		UserID:      r.UserID,
		SymbolID:    r.SymbolID,
		SymbolTitle: title,
		Count:       r.Count,
		PriceOpen:   price,
		StopLoss:    r.StopLoss,
		TakeProfit:  r.TakeProfit,
		IsBuy:       r.IsBuy,
	}, t)
	u.srv.muRep.Unlock()
	if err != nil {
		u.balance += sum
		u.srv.muRep.Lock()
		err = u.srv.rep.ChangeBalance(ctx, u.id, sum)
		u.srv.muRep.Unlock()
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
	u.positions[id] = &position
	allPositions, ok := u.allPositionsBySymbol[position.SymbolID]
	if !ok {
		u.allPositionsBySymbol[position.SymbolID] = make(map[int32]*model.Position)
		u.allPositionsBySymbol[position.SymbolID][position.ID] = &position
	} else {
		allPositions[position.ID] = &position
	}
	u.chPosByUser <- &request.PositionByUser{
		Action:     "ADD",
		UserID:     u.id,
		PositionID: position.ID,
	}
	return id, nil
}

func (u *User) ClosePosition(ctx context.Context, positionID int32) error {
	position, ok := u.positions[positionID]
	if !ok {
		return fmt.Errorf("you did not open a position with id %d", positionID)
	}
	var price float32
	if position.IsBuy == true {
		price = u.prices[position.SymbolID].Ask
	} else {
		price = u.prices[position.SymbolID].Bid
	}
	closePosition := request.ClosePosition{
		ID:         positionID,
		PriceClose: price,
	}
	sum := price * float32(position.Count)
	u.srv.muRep.Lock()
	err := u.srv.rep.ChangeBalance(ctx, u.id, sum)
	u.srv.muRep.Unlock()
	if err != nil {
		return err
	}
	u.balance += sum
	u.srv.muRep.Lock()
	err = u.srv.rep.ClosePosition(ctx, &closePosition)
	u.srv.muRep.Unlock()
	if err != nil {
		u.balance -= sum
		u.srv.muRep.Lock()
		err = u.srv.rep.ChangeBalance(ctx, u.id, -sum)
		u.srv.muRep.Unlock()
		if err != nil {
			log.Error(err)
		}
		return err
	}
	delete(u.positions, positionID)
	delete(u.allPositionsBySymbol[position.SymbolID], positionID)
	u.chPosByUser <- &request.PositionByUser{
		Action:     "DEL",
		UserID:     u.id,
		PositionID: positionID,
	}
	return nil
}

func (u *User) SetBalance(ctx context.Context, sum float32) error {
	u.srv.muRep.Lock()
	err := u.srv.rep.ChangeBalance(ctx, u.id, sum)
	u.srv.muRep.Unlock()
	if err != nil {
		return err
	}
	u.balance += sum
	return nil
}

func (u *User) GetBalance(ctx context.Context) float32 {
	return u.balance
}

// Return true if enough money and false if not enough money
func (u *User) checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}

func (u *User) close(positionID int32) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := u.ClosePosition(ctx, positionID)
	if err != nil {
		log.Error(err)
	}
}

// pnl is Profit and loss. Shows how much you earned or lost
func (u *User) pnl(positionID int32) (float32, error) {
	position := u.positions[positionID]
	price, ok := u.prices[position.SymbolID]
	if !ok {
		return 0, fmt.Errorf("pnl for position %d failed. isn't price. User is %d", positionID, u.id)
	}
	return price.Bid * float32(position.Count) - position.PriceOpen * float32(position.Count), nil
}

func stopLoss(position *model.Position, price *model.Price) bool {
	if position.IsBuy == true {
		return price.Ask <= position.StopLoss
	}
	return price.Bid >= position.StopLoss
}

func takeProfit(position *model.Position, price *model.Price) bool {
	if position.IsBuy == true {
		return price.Ask >= position.TakeProfit
	}
	return price.Bid <= position.TakeProfit
}
