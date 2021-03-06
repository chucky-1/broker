// Package server implements the server side of grpc
package server

import (
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/internal/service"
	"github.com/chucky-1/broker/protocol"
	log "github.com/sirupsen/logrus"

	"context"
	"fmt"
)

// Server contains methods of application on service side of grpc
type Server struct {
	protocol.UnimplementedBrokerServer
	srv *service.Service
}

// NewServer is constructor
func NewServer(srv *service.Service) *Server {
	return &Server{srv: srv}
}

// SignUp registers a new user
func (s *Server) SignUp(ctx context.Context, r *protocol.SignUpRequest) (*protocol.SignUpResponse, error) {
	id, err := s.srv.SignUp(ctx, r.Deposit)
	if err != nil {
		return nil, err
	}
	return &protocol.SignUpResponse{UserId: id}, nil
}

// SignIn logs into your account
func (s *Server) SignIn(ctx context.Context, r *protocol.SignInRequest) (*protocol.SignInResponse, error) {
	return &protocol.SignInResponse{}, nil
}

// OpenPosition opens a position
func (s *Server) OpenPosition(ctx context.Context, r *protocol.OpenPositionRequest) (*protocol.OpenPositionResponse, error) {
	positionID, err := s.srv.OpenPosition(ctx, &request.OpenPositionService{
		UserID:     r.UserId,
		SymbolID:   r.SymbolId,
		Price:      r.Price,
		Count:      r.Count,
		StopLoss:   r.StopLoss,
		TakeProfit: r.TakeProfit,
		IsBuy:      r.IsBuy,
	})
	if err != nil {
		if err.Error() == "user didn't find. Please, sign up" {
			return nil, err
		}
		if err.Error() == fmt.Sprintf("symbol with id %d didn't find", r.SymbolId) {
			return nil, err
		}
		if err.Error() == "not enough money" {
			return nil, err
		}
		if err.Error() == "price changed. Try again" {
			return nil, err
		}
		log.Error(err)
		return nil, err
	}
	return &protocol.OpenPositionResponse{PositionId: positionID}, nil
}

// ClosePosition closes a position
func (s *Server) ClosePosition(ctx context.Context, r *protocol.ClosePositionRequest) (*protocol.ClosePositionResponse, error) {
	err := s.srv.ClosePosition(ctx, r.PositionId)
	if err != nil {
		if err.Error() == fmt.Sprintf("you did not open a position with id %d", r.PositionId) {
			return nil, err
		}
		log.Error(err)
		return nil, err
	}
	return &protocol.ClosePositionResponse{}, nil
}

// SetBalance changes user's balance
func (s *Server) SetBalance(ctx context.Context, r *protocol.SetBalanceRequest) (*protocol.SetBalanceResponse, error) {
	err := s.srv.SetBalance(ctx, r.UserId, r.Sum)
	if err != nil {
		return nil, err
	}
	return &protocol.SetBalanceResponse{}, nil
}

// GetBalance returns user's balance
func (s *Server) GetBalance(ctx context.Context, r *protocol.GetBalanceRequest) (*protocol.GetBalanceResponse, error) {
	balance := s.srv.GetBalance(ctx, r.UserId)
	return &protocol.GetBalanceResponse{Sum: balance}, nil
}
