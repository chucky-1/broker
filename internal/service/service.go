// Package service have business logic
package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/internal/user"
	log "github.com/sirupsen/logrus"
	"time"

	"context"
	"errors"
	"fmt"
	"sync"
)

// Service implements business logic
type Service struct {
	muRep     sync.Mutex
	rep       *repository.Repository
	muSymbols sync.RWMutex
	symbols   map[int32]*model.Symbol // map[symbol.ID]*symbol
	muUsers   sync.RWMutex
	users     map[int32]*user.User // map[user.ID]*user
	chPrice   chan *model.Price
	muPrices  sync.RWMutex
	prices    map[int32]*model.Price
}

// NewService is constructor
func NewService(ctx context.Context, rep *repository.Repository, chPrice chan *model.Price,
	symbols map[int32]*model.Symbol) (*Service, error) {
	s := Service{
		rep:     rep,
		symbols: symbols,
		users:   make(map[int32]*user.User),
		chPrice: chPrice,
		prices:  make(map[int32]*model.Price),
	}
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- chPrice:
				s.muPrices.Lock()
				s.prices[price.ID] = price
				s.muPrices.Unlock()
				for _, u := range s.users {
					u.GetChanPrice() <- price
				}
			}
		}
	}(ctx)
	s.muRep.Lock()
	users, err := s.rep.GetAllUsers()
	s.muRep.Unlock()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		positions := make(map[int32]map[int32]*model.Position)
		openPositions, err := rep.GetOpenPositions(u.ID)
		for _, position := range openPositions {
			allPositions, ok := positions[position.SymbolID]
			if !ok {
				positions[position.SymbolID] = make(map[int32]*model.Position)
				positions[position.SymbolID][position.ID] = position
			} else {
				allPositions[position.ID] = position
			}
		}
		var closer request.PositionCloser = &s
		newUser, err := user.NewUser(ctx, u.ID, u.Balance, positions, closer)
		if err != nil {
			log.Error(err)
		} else {
			s.muUsers.Lock()
			s.users[newUser.GetID()] = newUser
			s.muUsers.Unlock()
		}
	}
	return &s, nil
}

// SignUp implements user registration
func (s *Service) SignUp(ctx context.Context, deposit float32) (int32, error) {
	s.muRep.Lock()
	u, err := s.rep.SignUp(ctx, deposit)
	s.muRep.Unlock()
	if err != nil {
		return 0, err
	}
	var closer request.PositionCloser = s
	newUser, err := user.NewUser(ctx, u.ID, u.Balance, make(map[int32]map[int32]*model.Position), closer)
	if err != nil {
		log.Error(err)
	} else {
		s.muUsers.Lock()
		s.users[newUser.GetID()] = newUser
		s.muUsers.Unlock()
	}
	return u.ID, nil
}

// OpenPosition opens position for user. Returns id of position
func (s *Service) OpenPosition(ctx context.Context, r *request.OpenPositionService) (int32, error) {
	s.muUsers.RLock()
	u, ok := s.users[r.UserID]
	s.muUsers.RUnlock()
	if !ok {
		return 0, errors.New("user didn't find. Please, sign up")
	}

	var price float32
	if r.IsBuy {
		s.muPrices.RLock()
		price = s.prices[r.SymbolID].Bid
		s.muPrices.RUnlock()
		ok = checkPrice(price, r.Price, r.IsBuy)
		if !ok {
			return 0, errors.New("price changed. Try again")
		}
	} else {
		s.muPrices.RLock()
		price = s.prices[r.SymbolID].Ask
		s.muPrices.RUnlock()
		ok = checkPrice(price, r.Price, r.IsBuy)
		if !ok {
			return 0, errors.New("price changed. Try again")
		}
	}
	currentBalance := u.GetBalance()
	sum := price * float32(r.Count)
	ok = checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}

	if r.IsBuy {
		s.muRep.Lock()
		err := s.rep.ChangeBalance(ctx, r.UserID, -sum)
		s.muRep.Unlock()
		if err != nil {
			return 0, err
		}
		u.ChangeBalance(-sum)
	} else {
		s.muRep.Lock()
		err := s.rep.ChangeBalance(ctx, r.UserID, sum)
		s.muRep.Unlock()
		if err != nil {
			return 0, err
		}
		u.ChangeBalance(sum)
	}

	t := time.Now()

	s.muSymbols.RLock()
	title := s.symbols[r.SymbolID].Title
	s.muSymbols.RUnlock()

	s.muRep.Lock()
	id, err := s.rep.OpenPosition(ctx, &request.OpenPositionRepository{
		UserID:      r.UserID,
		SymbolID:    r.SymbolID,
		SymbolTitle: title,
		Count:       r.Count,
		PriceOpen:   price,
		StopLoss:    r.StopLoss,
		TakeProfit:  r.TakeProfit,
		IsBuy:       r.IsBuy,
	}, t)
	s.muRep.Unlock()
	if err != nil {
		if r.IsBuy {
			u.ChangeBalance(sum)
			s.muRep.Lock()
			err = s.rep.ChangeBalance(ctx, u.GetID(), sum)
			s.muRep.Unlock()
			if err != nil {
				log.Error(err)
			}
		} else {
			u.ChangeBalance(-sum)
			s.muRep.Lock()
			err = s.rep.ChangeBalance(ctx, u.GetID(), -sum)
			s.muRep.Unlock()
			if err != nil {
				log.Error(err)
			}
		}
		return 0, err
	}

	position := model.Position {
		ID: id,
		UserID: r.UserID,
		SymbolID: r.SymbolID,
		SymbolTitle: title,
		Count: r.Count,
		PriceOpen: price,
		TimeOpen: t,
		StopLoss: r.StopLoss,
		TakeProfit: r.TakeProfit,
		IsBuy: r.IsBuy,
	}
	u.OpenPosition(&position)
	return id, nil
}

// ClosePosition closes position for user
func (s *Service) ClosePosition(ctx context.Context, positionID int32) error {
	s.muRep.Lock()
	userID, err := s.rep.GetUserIDByPositionID(ctx, positionID)
	s.muRep.Unlock()
	if err != nil {
		return fmt.Errorf("you did not open a position with id %d", positionID)
	}

	s.muUsers.RLock()
	u := s.users[userID]
	s.muUsers.RUnlock()

	s.muRep.Lock()
	position, err := s.rep.GetPosition(ctx, positionID)
	s.muRep.Unlock()
	if err != nil {
		return err
	}

	var price float32
	if position.IsBuy == true {
		s.muPrices.RLock()
		price = s.prices[position.SymbolID].Ask
		s.muPrices.RUnlock()
	} else {
		s.muPrices.RLock()
		price = s.prices[position.SymbolID].Bid
		s.muPrices.RUnlock()
	}
	sum := price * float32(position.Count)

	if position.IsBuy {
		s.muRep.Lock()
		err = s.rep.ChangeBalance(ctx, u.GetID(), sum)
		s.muRep.Unlock()
		if err != nil {
			return err
		}
		u.ChangeBalance(sum)
	} else {
		s.muRep.Lock()
		err = s.rep.ChangeBalance(ctx, u.GetID(), -sum)
		s.muRep.Unlock()
		if err != nil {
			return err
		}
		u.ChangeBalance(-sum)
	}

	s.muRep.Lock()
	err = s.rep.ClosePosition(ctx, &request.ClosePosition{
		ID:         positionID,
		PriceClose: price,
	})
	s.muRep.Unlock()
	if err != nil {
		if position.IsBuy {
			u.ChangeBalance(-sum)
			s.muRep.Lock()
			err = s.rep.ChangeBalance(ctx, u.GetID(), -sum)
			s.muRep.Unlock()
			if err != nil {
				log.Error(err)
			} else {
				u.ChangeBalance(sum)
				s.muRep.Lock()
				err = s.rep.ChangeBalance(ctx, u.GetID(), sum)
				s.muRep.Unlock()
				if err != nil {
					log.Error(err)
				}
			}
		}
		return err
	}
	u.ClosePosition(position.SymbolID, positionID)
	return nil
}

func (s *Service) Close(ctx context.Context, position *model.Position) error {
	var price float32
	if position.IsBuy {
		s.muRep.Lock()
		err := s.rep.ChangeBalance(ctx, position.UserID, position.AskClose * float32(position.Count))
		s.muRep.Unlock()
		if err != nil {
			return err
		}
		price = position.AskClose
	} else {
		s.muRep.Lock()
		err := s.rep.ChangeBalance(ctx, position.UserID, -(position.BidClose * float32(position.Count)))
		s.muRep.Unlock()
		if err != nil {
			return err
		}
		price = position.BidClose
	}
	s.muRep.Lock()
	err := s.rep.ClosePosition(ctx, &request.ClosePosition{
		ID:         position.ID,
		PriceClose: price,
	})
	s.muRep.Unlock()
	if err != nil {
		if position.IsBuy {
			s.muRep.Lock()
			err = s.rep.ChangeBalance(ctx, position.UserID, -(position.AskClose * float32(position.Count)))
			s.muRep.Unlock()
			if err != nil {
				log.Error(err)
			}
		} else {
			s.muRep.Lock()
			err = s.rep.ChangeBalance(ctx, position.UserID, position.BidClose * float32(position.Count))
			s.muRep.Unlock()
			if err != nil {
				log.Error(err)
			}
		}
		return err
	}
	return nil
}

// SetBalance changed balance of user
func (s *Service) SetBalance(ctx context.Context, userID int32, sum float32) error {
	s.muUsers.RLock()
	u := s.users[userID]
	s.muUsers.RUnlock()

	s.muRep.Lock()
	err := s.rep.ChangeBalance(ctx, userID, sum)
	s.muRep.Unlock()
	if err != nil {
		return err
	}
	u.ChangeBalance(sum)
	return nil
}

// GetBalance returns balance of user
func (s *Service) GetBalance(ctx context.Context, userID int32) float32 {
	s.muUsers.RLock()
	u := s.users[userID]
	s.muUsers.RUnlock()
	return u.GetBalance()
}

func checkPrice(priceActual, priceWait float32, isBuy bool) bool {
	if isBuy {
		return priceWait >= priceActual
	}
	return priceWait <= priceActual
}

// Return true if enough money and false if not enough money
func checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}
