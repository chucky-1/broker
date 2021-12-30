// Package service have business logic
package service

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
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
	users          map[int32]*User // map[user.ID]*user
	muPosByUser    sync.RWMutex
	positionByUser map[int32]int32 // map[position.ID]user.ID
	chPosByUser    chan *request.PositionByUser
	chPrice        chan *model.Price
}

// NewService is constructor
func NewService(ctx context.Context, rep *repository.Repository, chPrice chan *model.Price, symbols map[int32]*model.Symbol) (*Service, error) {
	s := Service{
		rep:            rep,
		symbols:        symbols,
		users:          make(map[int32]*User),
		positionByUser: make(map[int32]int32),
		chPosByUser:    make(chan *request.PositionByUser),
		chPrice: chPrice,
	}
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- chPrice:
				for _, user := range s.users {
					user.chPrice <- price
				}
			}
		}
	}(ctx)
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case positionByUser := <- s.chPosByUser:
				switch {
				case positionByUser.Action == "ADD":
					s.muPosByUser.Lock()
					s.positionByUser[positionByUser.PositionID] = positionByUser.UserID
					s.muPosByUser.Unlock()
				case positionByUser.Action == "DEL":
					s.muPosByUser.Lock()
					delete(s.positionByUser, positionByUser.PositionID)
					s.muPosByUser.Unlock()
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
	for _, user := range users {
		newUser, err := NewUser(ctx, &s, user.ID, user.Balance, s.chPosByUser)
		if err != nil {
			log.Error(err)
		} else {
			s.muUsers.Lock()
			s.users[newUser.id] = newUser
			s.muUsers.Unlock()
		}
	}
	return &s, nil
}

func (s *Service) SignUp(ctx context.Context, deposit float32) (int32, error) {
	s.muRep.Lock()
	user, err := s.rep.SignUp(ctx, deposit)
	s.muRep.Unlock()
	if err != nil {
		return 0, err
	}
	newUser, err := NewUser(ctx, s, user.ID, user.Balance, s.chPosByUser)
	if err != nil {
		log.Error(err)
	} else {
		s.muUsers.Lock()
		s.users[newUser.id] = newUser
		s.muUsers.Unlock()
	}
	return user.ID, nil
}

func (s *Service) OpenPosition(ctx context.Context, request *request.OpenPositionService) (int32, error) {
	s.muUsers.RLock()
	user, ok := s.users[request.UserID]
	s.muUsers.RUnlock()
	if !ok {
		return 0, errors.New("user didn't find. Please, sign up")
	}
	return user.OpenPosition(ctx, request)
}

func (s *Service) ClosePosition(ctx context.Context, positionID int32) error {
	s.muPosByUser.RLock()
	userID, ok := s.positionByUser[positionID]
	s.muPosByUser.RUnlock()
	if !ok {
		return fmt.Errorf("you did not open a position with id %d", positionID)
	}

	s.muUsers.RLock()
	user := s.users[userID]
	s.muUsers.RUnlock()
	return user.ClosePosition(ctx, positionID)
}

func (s *Service) SetBalance(ctx context.Context, userID int32, sum float32) error {
	s.muUsers.RLock()
	user := s.users[userID]
	s.muUsers.RUnlock()
	return user.SetBalance(ctx, sum)
}

func (s *Service) GetBalance(ctx context.Context, userID int32) float32 {
	s.muUsers.RLock()
	user := s.users[userID]
	s.muUsers.RUnlock()
	return user.GetBalance(ctx)
}
