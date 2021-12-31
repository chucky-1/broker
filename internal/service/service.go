// Package service have business logic
package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/internal/user"
	log "github.com/sirupsen/logrus"

	"context"
	"errors"
	"fmt"
	"sync"
)

// Service implements business logic
type Service struct {
	muRep          sync.Mutex
	rep            *repository.Repository
	muSymbols      sync.RWMutex
	symbols        map[int32]*model.Symbol // map[symbol.ID]*symbol
	muUsers        sync.RWMutex
	users          map[int32]*user.User // map[user.ID]*user
	chPrice        chan *model.Price
	muPrices       sync.RWMutex
	prices         map[int32]*model.Price
}

// NewService is constructor
func NewService(ctx context.Context, rep *repository.Repository, chPrice chan *model.Price, symbols map[int32]*model.Symbol) (*Service, error) {
	s := Service{
		rep:            rep,
		symbols:        symbols,
		users:          make(map[int32]*user.User),
		chPrice:        chPrice,
		prices:         make(map[int32]*model.Price),
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
		newUser, err := user.NewUser(ctx, &s, u.ID, u.Balance)
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

func (s *Service) SignUp(ctx context.Context, deposit float32) (int32, error) {
	s.muRep.Lock()
	u, err := s.rep.SignUp(ctx, deposit)
	s.muRep.Unlock()
	if err != nil {
		return 0, err
	}
	newUser, err := user.NewUser(ctx, s, u.ID, u.Balance)
	if err != nil {
		log.Error(err)
	} else {
		s.muUsers.Lock()
		s.users[newUser.GetID()] = newUser
		s.muUsers.Unlock()
	}
	return u.ID, nil
}

func (s *Service) OpenPosition(ctx context.Context, request *request.OpenPositionService) (int32, error) {
	s.muUsers.RLock()
	u, ok := s.users[request.UserID]
	s.muUsers.RUnlock()
	if !ok {
		return 0, errors.New("user didn't find. Please, sign up")
	}
	return u.OpenPosition(ctx, request)
}

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
	return u.ClosePosition(ctx, positionID)
}

func (s *Service) SetBalance(ctx context.Context, userID int32, sum float32) error {
	s.muUsers.RLock()
	u := s.users[userID]
	s.muUsers.RUnlock()
	return u.SetBalance(ctx, sum)
}

func (s *Service) GetBalance(ctx context.Context, userID int32) float32 {
	s.muUsers.RLock()
	u := s.users[userID]
	s.muUsers.RUnlock()
	return u.GetBalance()
}

func (s *Service) GetRepository() (*repository.Repository, *sync.Mutex) {
	return s.rep, &s.muRep
}

func (s *Service) GetSymbols() (map[int32]*model.Symbol, *sync.RWMutex) {
	return s.symbols, &s.muSymbols
}

func (s *Service) GetChanPrice() chan *model.Price {
	return s.chPrice
}

func (s *Service) GetPrices() (map[int32]*model.Price, *sync.RWMutex) {
	return s.prices, &s.muPrices
}
