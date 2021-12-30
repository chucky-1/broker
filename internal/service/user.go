package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"

	"context"
	"errors"
	"time"
)

type User struct {
	srv         *Service
	id          int32
	balance     float32
	chPrice     chan *model.Price
	positions   map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
}

func NewUser(ctx context.Context, srv *Service, id int32, balance float32) (*User, error) {
	u := User{
		srv:         srv,
		id:          id,
		balance:     balance,
		chPrice:     make(chan *model.Price),
		positions:   make(map[int32]map[int32]*model.Position),
	}

	srv.muRep.Lock()
	positions, err := srv.rep.GetOpenPositions(u.id)
	srv.muRep.Unlock()
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
					p, err := pnl(position, price)
					if err != nil {
						log.Error(err)
						continue
					}
					log.Infof("pnl for position %d is %f", position.ID, p)
					if stopLoss(position, price) {
						u.close(position.ID)
					}
					if takeProfit(position, price) {
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
		u.srv.muPrices.RLock()
		price = u.srv.prices[r.SymbolID].Bid
		u.srv.muPrices.RUnlock()
		ok := checkPrice(price, r.Price, r.IsBuy)
		if !ok {
			return 0, errors.New("price changed. Try again")
		}
	} else {
		u.srv.muPrices.RLock()
		price = u.srv.prices[r.SymbolID].Ask
		u.srv.muPrices.RUnlock()
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
	allPositions, ok := u.positions[position.SymbolID]
	if !ok {
		u.positions[position.SymbolID] = make(map[int32]*model.Position)
		u.positions[position.SymbolID][position.ID] = &position
	} else {
		allPositions[position.ID] = &position
	}
	return id, nil
}

func (u *User) ClosePosition(ctx context.Context, positionID int32) error {
	u.srv.muRep.Lock()
	position, err := u.srv.rep.GetPosition(ctx, positionID)
	u.srv.muRep.Unlock()
	if err != nil {
		return err
	}
	var price float32
	if position.IsBuy == true {
		u.srv.muPrices.RLock()
		price = u.srv.prices[position.SymbolID].Ask
		u.srv.muPrices.RUnlock()
	} else {
		u.srv.muPrices.RLock()
		price = u.srv.prices[position.SymbolID].Bid
		u.srv.muPrices.RUnlock()
	}
	closePosition := request.ClosePosition{
		ID:         positionID,
		PriceClose: price,
	}
	sum := price * float32(position.Count)
	u.srv.muRep.Lock()
	err = u.srv.rep.ChangeBalance(ctx, u.id, sum)
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
	delete(u.positions[position.SymbolID], positionID)
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
func pnl(position *model.Position, price *model.Price) (float32, error) {
	return price.Bid * float32(position.Count) - position.PriceOpen * float32(position.Count), nil
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
