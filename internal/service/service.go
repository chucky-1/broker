// Package service have business logic
package service

import (
	"errors"
	"fmt"
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"
	"time"
)

// Service is unique for each user
type Service struct {
	rep       *repository.Repository
	users     map[int32]*model.User
	chPrice   chan *model.Price
	prices    map[int32]*model.Price    // map[price.ID]*price
	positions map[int32]*model.Position // map[position.ID]*position
	allPositionsBySymbol map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
	symbols   map[int32]*model.Symbol
}

// NewService is constructor
func NewService(rep *repository.Repository, chPrice chan *model.Price, symbols map[int32]*model.Symbol) (*Service, error) {
	s := Service{
		rep: rep,
		users: make(map[int32]*model.User),
		chPrice: chPrice,
		prices: make(map[int32]*model.Price),
		positions: make(map[int32]*model.Position),
		allPositionsBySymbol: make(map[int32]map[int32]*model.Position),
		symbols: symbols,
	}
	go func() {
		for {
			price := <- chPrice
			s.prices[price.ID] = price
			go func() {
				for _, position := range s.allPositionsBySymbol[price.ID] {
					pnl := s.pnl(position.ID)
					log.Infof("pnl for position %d is %f", position.ID, pnl)
					p, ok := stopLoss(position, s.prices[position.ID])
					if ok {
						err := s.rep.ClosePosition(&request.ClosePosition{
							ID:         position.ID,
							PriceClose: p,
						})
						if err != nil {
							log.Error(err)
						} else {
							delete(s.positions, position.ID)
							delete(s.allPositionsBySymbol[position.SymbolID], position.ID)
						}
					}
					p, ok = takeProfit(position, s.prices[position.ID])
					if ok {
						err := s.rep.ClosePosition(&request.ClosePosition{
							ID:         position.ID,
							PriceClose: p,
						})
						if err != nil {
							log.Error(err)
						} else {
							delete(s.positions, position.ID)
							delete(s.allPositionsBySymbol[position.SymbolID], position.ID)
						}
					}
				}
			}()
		}
	}()
	users, err := s.rep.GetAllUsers()
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		s.users[user.ID] = user
	}
	positions, err := s.rep.GetAllOpenPositions()
	if err != nil {
		return nil, err
	}
	for _, position := range positions {
		s.positions[position.ID] = position
		allPositions, ok := s.allPositionsBySymbol[position.SymbolID]
		if !ok {
			s.allPositionsBySymbol[position.SymbolID] = make(map[int32]*model.Position)
			s.allPositionsBySymbol[position.SymbolID][position.ID] = position
		} else {
			allPositions[position.ID] = position
		}
	}
	return &s, nil
}

func (s *Service) SignUp(deposit float32) (int32, error) {
	user, err := s.rep.SignUp(deposit)
	if err != nil {
		return 0, err
	}
	s.users[user.ID] = user
	return user.ID, nil
}

func (s *Service) SignIn() {
	return
}

func (s *Service) OpenPosition(r *request.OpenPositionService) (int32, error) {
	var price float32
	if r.IsBuy == true {
		price = s.prices[r.SymbolID].Bid
	} else {
		price = s.prices[r.SymbolID].Ask
	}
	currentBalance := s.users[r.UserID].Balance
	sum := price * float32(r.Count)
	ok := s.checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}
	err := s.rep.ChangeBalance(r.UserID, -sum)
	if err != nil {
		return 0, err
	}
	s.users[r.UserID].Balance -= sum

	t := time.Now()

	id, err := s.rep.OpenPosition(&request.OpenPositionRepository{
		UserID:      r.UserID,
		SymbolID:    r.SymbolID,
		SymbolTitle: s.symbols[r.SymbolID].Title,
		Count:       r.Count,
		PriceOpen:   price,
		StopLoss:    r.StopLoss,
		TakeProfit:  r.TakeProfit,
		IsBuy:       r.IsBuy,
	}, t)
	if err != nil {
		s.users[r.UserID].Balance += sum
		err = s.rep.ChangeBalance(s.users[r.UserID].ID, sum)
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
	s.positions[id] = &position
	return id, nil
}

func (s *Service) ClosePosition(positionID int32) error {
	position, ok := s.positions[positionID]
	if !ok {
		return fmt.Errorf("you did not open a position with id %d", positionID)
	}
	var price float32
	if position.IsBuy == true {
		price = s.prices[position.SymbolID].Ask
	} else {
		price = s.prices[position.SymbolID].Bid
	}
	closePosition := request.ClosePosition{
		ID:         positionID,
		PriceClose: price,
	}
	sum := price * float32(position.Count)
	err := s.rep.ChangeBalance(s.users[position.UserID].ID, sum)
	if err != nil {
		return err
	}
	s.users[position.UserID].Balance += sum
	err = s.rep.ClosePosition(&closePosition)
	if err != nil {
		s.users[position.UserID].Balance -= sum
		err = s.rep.ChangeBalance(s.users[position.UserID].ID, -sum)
		if err != nil {
			log.Error(err)
		}
		return err
	}
	delete(s.positions, positionID)
	return nil
}

func (s *Service) SetBalance(userID int32, sum float32) error {
	err := s.rep.ChangeBalance(userID, sum)
	if err != nil {
		return err
	}
	s.users[userID].Balance += sum
	return nil
}

func (s *Service) GetBalance(userID int32) float32 {
	return s.users[userID].Balance
}

// Return true if enough money and false if not enough money
func (s *Service) checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}

// pnl is Profit and loss. Shows how much you earned or lost
func (s *Service) pnl(positionID int32) float32 {
	position := s.positions[positionID]
	price := s.prices[position.ID]
	return price.Bid * float32(position.Count) - position.PriceOpen * float32(position.Count)
}

func stopLoss(position *model.Position, price *model.Price) (float32, bool) {
	if position.IsBuy == true {
		return price.Ask, price.Ask <= position.StopLoss
	}
	return price.Bid, price.Bid >= position.StopLoss
}

func takeProfit(position *model.Position, price *model.Price) (float32, bool) {
	if position.IsBuy == true {
		return price.Ask, price.Ask >= position.TakeProfit
	}
	return price.Bid, price.Bid <= position.TakeProfit
}
