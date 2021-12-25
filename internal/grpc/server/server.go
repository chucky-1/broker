// Package server implements the server side of grpc
package server

import (
	"github.com/chucky-1/broker/internal/service"
	"github.com/chucky-1/broker/protocol"
	log "github.com/sirupsen/logrus"

	"errors"
	"fmt"
)

// Server contains methods of application on service side of grpc
type Server struct {
	protocol.UnimplementedPositionsServer
	chanUserID  chan int32
	chanDeposit chan float32
	chanService chan *service.Service
}

// NewServer is constructor
func NewServer(chanUserID chan int32, chanDeposit chan float32, chanService chan *service.Service) *Server {
	return &Server{chanUserID: chanUserID, chanDeposit: chanDeposit, chanService: chanService}
}

// Position listens commands from client and does work
func (s *Server) Position(stream protocol.Positions_PositionServer) error {
	recv, err := stream.Recv()
	if err != nil {
		log.Error(err)
		return errors.New("initialization error")
	}
	if recv.Act != "INIT" {
		return errors.New("initialization error")
	}

	s.chanUserID <- recv.UserId
	s.chanDeposit <- recv.Deposit
	srv := <-s.chanService

	err = stream.Send(&protocol.Response{
		Act:     "INIT",
		Message: "You have successfully logged into the system",
	})
	if err != nil {
		log.Error(err)
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			recv, err = stream.Recv()
			if err != nil {
				log.Error(err)
				continue
			}
			switch {
			case recv.Act == "OPEN":
				positionID, err := srv.Open(recv.StockId, recv.Count)
				if err != nil {
					if err.Error() == fmt.Sprintf("stock with id %d didn't find", recv.StockId) {
						errSend := stream.Send(&protocol.Response{
							Act:     "OPEN",
							Message: err.Error(),
						})
						if errSend != nil {
							log.Error(errSend)
						}
						continue
					}
					if err.Error() == "not enough money" {
						errSend := stream.Send(&protocol.Response{
							Act:     "OPEN",
							Message: err.Error(),
						})
						if errSend != nil {
							log.Error(errSend)
						}
						continue
					}
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
						Act:     "OPEN",
						Message: "Position didn't open. Try else!",
					})
					if errSend != nil {
						log.Error(errSend)
					}
				} else {
					err = stream.Send(&protocol.Response{
						Act:        "OPEN",
						Message:    "Position successfully opened",
						PositionId: positionID,
					})
					if err != nil {
						log.Error(err)
					}
				}
			case recv.Act == "CLOSE":
				err = srv.Close(recv.PositionId)
				if err != nil {
					if err.Error() == fmt.Sprintf("you did not open a position with id %d", recv.PositionId) {
						errSend := stream.Send(&protocol.Response{
							Act:        "CLOSE",
							Message:    err.Error(),
							PositionId: recv.PositionId,
						})
						if errSend != nil {
							log.Error(errSend)
						}
						continue
					}
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
						Act:        "CLOSE",
						Message:    "Position didn't close",
						PositionId: recv.PositionId,
					})
					if errSend != nil {
						log.Error(errSend)
					}
				} else {
					err = stream.Send(&protocol.Response{
						Act:        "CLOSE",
						Message:    "Position successfully closed",
						PositionId: recv.PositionId,
					})
					if err != nil {
						log.Error(err)
					}
				}
			case recv.Act == "BALANCE":
				balance := srv.GetBalance()
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
