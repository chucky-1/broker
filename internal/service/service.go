// Package service have business logic
package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/protocol"
	log "github.com/sirupsen/logrus"

	"errors"
	"fmt"
	"strconv"
	"time"
)

// Service is unique for each user
type Service struct {
	rep       *repository.Repository
	user      *model.User
	grpcID    string
	positions map[int32]*model.Position // map[model.Position.ID]*model.Position
	count     map[int32]int             // map[stock.ID]count. It is the total number of shares for different positions
	stocks    map[int32]*protocol.Stock // map[stock.ID]*stock. Ih has the current stocks
	ch        chan *protocol.Stock      // Current prices come here
	chnPos    chan *request.Position
}

// NewService is constructor
func NewService(rep *repository.Repository, userID int32, grpcID string, deposit float32, chnPos chan *request.Position) (*Service, error) {
	s := Service{
		rep:    rep,
		grpcID: grpcID,
		count:  make(map[int32]int),
		stocks: make(map[int32]*protocol.Stock),
		ch:     make(chan *protocol.Stock),
		chnPos: chnPos,
	}
	go func() {
		for stock := range s.ch {
			s.stocks[stock.Id] = stock
		}
	}()

	var user *model.User
	user, err := s.rep.SignIn(userID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			user, err = s.rep.CreateUser(deposit)
			if err != nil {
				return nil, errors.New("initialization error")
			}
		} else {
			return nil, errors.New("initialization error")
		}
	}
	s.user = user

	positions, err := s.rep.GetOpenPositions(s.user.ID)
	if err != nil {
		return nil, err
	}
	s.positions = positions
	for _, position := range s.positions {
		num, ok := s.count[position.StockID]
		if !ok {
			s.count[position.StockID] = 1
		} else {
			num++
		}
	}
	return &s, nil
}

// CreateUser creates a new user
func (s *Service) CreateUser(deposit float32) (*model.User, error) {
	return s.rep.CreateUser(deposit)
}

// SignIn returns user
func (s *Service) SignIn(id int32) (*model.User, error) {
	return s.rep.SignIn(id)
}

// Open creates a new position
func (s *Service) Open(stockID, count int32, stopLoss, takeProfit float32) (int32, error) {
	_, ok := s.stocks[stockID]
	if !ok {
		return 0, fmt.Errorf("stock with id %d didn't find", stockID)
	}
	position := model.Position{
		UserID:     s.user.ID,
		StockID:    stockID,
		StockTitle: s.stocks[stockID].Title,
		Count:      count,
		PriceOpen:  s.stocks[stockID].Price,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	}
	currentBalance := s.GetBalance()
	sum := position.PriceOpen * float32(count)
	ok = s.checkTransaction(currentBalance, sum)
	if !ok {
		return 0, errors.New("not enough money")
	}
	err := s.rep.ChangeBalance(s.user.ID, -sum)
	if err != nil {
		return 0, err
	}
	s.user.Balance -= sum
	t, err := getTime(s.stocks[stockID].Update)
	if err != nil {
		log.Error(err)
	}

	id, err := s.rep.Open(&position, t)
	if err != nil {
		s.user.Balance += sum
		err = s.rep.ChangeBalance(s.user.ID, sum)
		if err != nil {
			log.Error(err)
		}
		return 0, err
	}

	position.ID = id
	s.positions[id] = &position
	num, ok := s.count[position.StockID]
	if !ok {
		s.count[position.StockID] = 1
	} else {
		num++
	}

	s.chnPos <- &request.Position{
		Act:        "OPEN",
		PositionID: position.ID,
		StockID:    position.StockID,
		UserID:     position.UserID,
		Count:      position.Count,
		Price:      position.PriceOpen,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	}
	return position.ID, nil
}

// Close closes the position
func (s *Service) Close(positionID int32) error {
	_, ok := s.positions[positionID]
	if !ok {
		return fmt.Errorf("you did not open a position with id %d", positionID)
	}
	position := request.ClosePosition{
		ID:         positionID,
		PriceClose: s.stocks[s.positions[positionID].StockID].Price,
	}
	sum := position.PriceClose * float32(s.positions[positionID].Count)
	err := s.rep.ChangeBalance(s.user.ID, sum)
	if err != nil {
		return err
	}
	s.user.Balance += sum
	err = s.rep.Close(&position)
	if err != nil {
		s.user.Balance -= sum
		err = s.rep.ChangeBalance(s.user.ID, -sum)
		if err != nil {
			log.Error(err)
		}
		return err
	}

	stockID := s.positions[positionID].StockID
	num := s.count[stockID]
	if num == 1 {
		delete(s.count, stockID)
	} else {
		num--
	}
	delete(s.positions, positionID)
	s.chnPos <- &request.Position{
		Act:        "CLOSE",
		PositionID: position.ID,
		StockID:    stockID,
		//Count:    s.positions[positionID].Count,
		Price:      position.PriceClose,
	}
	return nil
}

// GetBalance returns user's balance
func (s *Service) GetBalance() float32 {
	return s.user.Balance
}

// GetCount returns the count
func (s *Service) GetCount() map[int32]int {
	return s.count
}

// GetChan returns the chan
func (s *Service) GetChan() chan *protocol.Stock {
	return s.ch
}

// GetUser returns the user
func (s *Service) GetUser() *model.User {
	return s.user
}

// pnl is Profit and loss. Shows how much you earned or lost
func (s *Service) pnl(positionID int32) float32 {
	position := s.positions[positionID]
	stock := s.stocks[position.StockID]
	return stock.Price * float32(position.Count) - position.PriceOpen * float32(position.Count)
}

// Return true if enough money and false if not enough money
func (s *Service) checkTransaction(balance, sum float32) bool {
	return balance - sum >= 0
}

// getTime formats a string to a date. String from Redis ID
func getTime(id string) (time.Time, error) {
	mkr, err := strconv.Atoi(id)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Unix(int64(mkr)/1000, 0)
	return t, nil
}
