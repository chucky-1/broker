// Package server implements the server side of grpc
package server

import (
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/protocol"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"sync"
)

// Server contains methods of application on service side of grpc
type Server struct {
	protocol.UnimplementedSwopsServer
	rep *repository.Repository
	ch  chan *protocol.Stock
	mu  sync.RWMutex
	num int32 // total number of clients
}

// NewServer is constructor
func NewServer(rep *repository.Repository, ch chan *protocol.Stock) *Server {
	return &Server{rep: rep, ch: ch, num: 0}
}

// Swop listens commands from client and does work
func (s *Server) Swop(stream protocol.Swops_SwopServer) error {
	s.mu.Lock()
	s.num++
	s.mu.Unlock()
	grpcID := uuid.New().String()
	internalChan := make(chan *protocol.Application)
	go func() {
		for {
			select {
			case <-stream.Context().Done():
				return
			default:
				recv, err := stream.Recv()
				if err != nil {
					log.Error(err)
					continue
				} else {
					internalChan <- recv
				}
			}
		}
	}()
	err := s.rep.CreateUser(grpcID)
	if err != nil {
		return err
	}
	for {
		select {
		case <-stream.Context().Done():
			s.mu.Lock()
			s.num--
			s.mu.Unlock()
			return stream.Context().Err()
		case stock := <-s.ch:
			swops, err := s.rep.GetOpenSwops(grpcID)
			if len(swops) == 0 {
				continue
			}
			if err != nil {
				log.Error(err)
				err = stream.Send(&protocol.Response{
					Act:     "BALANCE_REAL_TIME",
					Message: "Your balance is no longer up to date. Server errors",
				})
				if err != nil {
					log.Error(err)
				}
				continue
			}
			for _, swop := range swops {
				if swop.StockID != stock.Id {
					continue
				}
				value, details, err := s.rep.GetBalanceRealTime(swops)
				if err != nil {
					log.Error(err)
					err = stream.Send(&protocol.Response{
						Act:     "BALANCE_REAL_TIME",
						Message: "Your balance is no longer up to date. Server errors",
					})
					if err != nil {
						log.Error(err)
					}
					break
				}
				protoDetails := make([]*protocol.Detail, len(details))
				for _, detail := range details {
					protoDetail := protocol.Detail{
						Stock: &protocol.Stock{
							Id: detail.Stock.ID,
							Title: detail.Stock.Title,
							Price: detail.Stock.Price,
							Update: detail.Stock.Update,
						},
						Count: detail.Count,
					}
					protoDetails = append(protoDetails, &protoDetail)
				}
				err = stream.Send(&protocol.Response{
					Act: "BALANCE_REAL_TIME",
					BalanceRealTime: value,
					Details: protoDetails,
				})
				if err != nil {
					log.Error(err)
					err = stream.Send(&protocol.Response{
						Act:     "BALANCE_REAL_TIME",
						Message: "Your balance is no longer up to date. Server errors",
					})
					if err != nil {
						log.Error(err)
					}
				}
				break
			}
		case recv := <-internalChan:
			switch {
			case recv.Act == "OPEN":
				swopID, err := s.rep.Open(grpcID, recv.StockId, recv.Count)
				if err != nil {
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
						Act: "OPEN",
						Message: "Position didn't open. Try else!",
					})
					if errSend != nil {
						log.Error(errSend)
					}
				} else {
					err = stream.Send(&protocol.Response{
						Act:     "OPEN",
						Message: "Position successfully opened",
						SwopId:  int32(swopID),
					})
					if err != nil {
						log.Error(err)
					}
				}
			case recv.Act == "CLOSE":
				err = s.rep.Close(grpcID, recv.SwopId)
				if err != nil {
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
						Act:     "CLOSE",
						Message: "Position didn't close",
						SwopId:  recv.SwopId,
					})
					if errSend != nil {
						log.Error(errSend)
					}
				} else {
					err = stream.Send(&protocol.Response{
						Act:     "CLOSE",
						Message: "Position successfully closed",
						SwopId:  recv.SwopId,
					})
					if err != nil {
						log.Error(err)
					}
				}
			case recv.Act == "BALANCE":
				balance, err := s.rep.GetBalance(grpcID)
				if err != nil {
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
						Act:     "BALANCE",
						Message: "Balance unknown",
					})
					if errSend != nil {
						log.Error(errSend)
					}
				} else {
					err = stream.Send(&protocol.Response{
						Act:     "BALANCE",
						Message: "Balance received successfully",
						Balance: balance,
					})
					if err != nil {
						log.Error(err)
					}
				}
			}
		}
	}
}

func (s *Server)GetNum() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.num
}
