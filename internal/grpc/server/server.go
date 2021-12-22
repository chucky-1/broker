package server

import (
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/protocol"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Server contains methods of application on service side of grpc
type Server struct {
	protocol.UnimplementedSwopsServer
	rep *repository.Repository
}

// NewServer is constructor
func NewServer(rep *repository.Repository) *Server {
	return &Server{rep: rep}
}

// Swop listens commands from client and does work
func (s *Server) Swop(stream protocol.Swops_SwopServer) error {
	grpcID := uuid.New().String()
	err := s.rep.CreateUser(grpcID)
	if err != nil {
		return err
	}
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			recv, err := stream.Recv()
			if err != nil {
				return err
			}
			switch {
			case recv.Act == "OPEN":
				swopID, err := s.rep.Open(grpcID, recv.StockId, recv.Count)
				if err != nil {
					log.Error(err)
					errSend := stream.Send(&protocol.Response{
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
