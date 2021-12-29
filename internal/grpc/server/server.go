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

func (s *Server)SignUp(ctx context.Context, r *protocol.SignUpRequest) (*protocol.SignUpResponse, error) {
	id, err := s.srv.SignUp(r.Deposit)
	if err != nil {
		return nil, err
	}
	return &protocol.SignUpResponse{UserId: id}, nil
}

func (s *Server)SignIn(ctx context.Context, r *protocol.SignInRequest) (*protocol.SignInResponse, error) {
	return &protocol.SignInResponse{}, nil
}

func (s *Server) OpenPosition(ctx context.Context, r *protocol.OpenPositionRequest) (*protocol.OpenPositionResponse, error) {
	positionID, err := s.srv.OpenPosition(&request.OpenPositionService{
		UserID:     r.UserId,
		SymbolID:   r.SymbolId,
		Count:      r.Count,
		StopLoss:   r.StopLoss,
		TakeProfit: r.TakeProfit,
		IsBuy:      r.IsBuy,
	})
	if err != nil {
		if err.Error() == fmt.Sprintf("symbol with id %d didn't find", r.SymbolId) {
			return nil, err
		}
		if err.Error() == "not enough money" {
			return nil, err
		}
		log.Error(err)
		return nil, err
	}
	return &protocol.OpenPositionResponse{PositionId: positionID}, nil
}

func (s *Server) ClosePosition(ctx context.Context, request *protocol.ClosePositionRequest) (*protocol.ClosePositionResponse, error) {
	err := s.srv.ClosePosition(request.PositionId)
	if err != nil {
		if err.Error() == fmt.Sprintf("you did not open a position with id %d", request.PositionId) {
			return nil, err
		}
		log.Error(err)
		return nil, err
	}
	return &protocol.ClosePositionResponse{}, nil
}

func (s *Server) SetBalance(ctx context.Context, request *protocol.SetBalanceRequest) (*protocol.SetBalanceResponse, error) {
	err := s.srv.SetBalance(request.UserId, request.Sum)
	if err != nil {
		return nil, err
	}
	return &protocol.SetBalanceResponse{}, nil
}

func (s *Server) GetBalance(ctx context.Context, request *protocol.GetBalanceRequest) (*protocol.GetBalanceResponse, error) {
	balance := s.srv.GetBalance(request.UserId)
	return &protocol.GetBalanceResponse{Sum: balance}, nil
}
