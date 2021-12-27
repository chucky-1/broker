package service

import (
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	"github.com/chucky-1/broker/protocol"
	log "github.com/sirupsen/logrus"
)

// Bot analyzes information on all open positions
type Bot struct {
	rep       *repository.Repository
	services  map[string]*Service // map[grpcID]*service
	chSrvAdd  chan *Service
	chSrvDel  chan string // grpcID
	positions map[int32]map[int32]*request.Position // map[stock.ID]map[position.ID]*position
	chPos     chan *request.Position
	stocks    map[int32]*protocol.Stock
	chStocks  chan *protocol.Stock
}

// NewBot is struct
func NewBot(rep *repository.Repository, chSrvAdd chan *Service, chSrvDel chan string, chPos chan *request.Position, chStocks chan *protocol.Stock) error {
	b := &Bot{
		rep:       rep,
		services:  make(map[string]*Service),
		chSrvAdd:  chSrvAdd,
		chSrvDel:  chSrvDel,
		positions: make(map[int32]map[int32]*request.Position),
		chPos:     chPos,
		stocks:    make(map[int32]*protocol.Stock),
		chStocks:  chStocks,
	}
	positions, err := rep.GetAllOpenPositions()
	if err != nil {
		return err
	}
	for _, position := range positions {
		req := &request.Position{
			Act:        "INIT",
			PositionID: position.PositionID,
			StockID:    position.StockID,
			Count:      position.Count,
			Price:      position.PriceOpen,
			StopLoss:   position.StopLoss,
			TakeProfit: position.TakeProfit,
		}
		m, ok := b.positions[position.StockID]
		if !ok {
			b.positions[position.StockID] = make(map[int32]*request.Position)
			b.positions[position.StockID][position.PositionID] = req
		} else {
			m[position.PositionID] = req
		}
	}
	go func() {
		for srv := range b.chSrvAdd {
			b.services[srv.grpcID] = srv
		}
	}()
	go func() {
		for grpcID := range b.chSrvDel {
			delete(b.services, grpcID)
		}
	}()
	go func() {
		for position := range chPos {
			if position.Act == "OPEN" {
				m, ok := b.positions[position.StockID]
				if !ok {
					b.positions[position.StockID] = make(map[int32]*request.Position)
					b.positions[position.StockID][position.PositionID] = position
				} else {
					m[position.PositionID] = position
				}
			}
			if position.Act == "CLOSE" {
				delete(b.positions[position.StockID], position.PositionID)
			}
		}
	}()
	go func() {
		for stock := range b.chStocks {
			b.stocks[stock.Id] = stock
			cStock := stock
			go func() {
				for _, position := range b.positions[cStock.Id] {
					pnl := b.pnl(position.StockID, position.PositionID)
					log.Infof("pnl for position ib %d is %f", position.PositionID, pnl)
					switch {
					case b.stopLoss(position, cStock):

					case b.takeProfit(position, cStock):

					}
				}
			}()
		}
	}()
	return nil
}

// pnl is Profit and loss. Shows how much you earned or lost
func (b *Bot) pnl(stockID, positionID int32) float32 {
	position := b.positions[stockID][positionID]
	stock := b.stocks[stockID]
	return stock.Price * float32(position.Count) - position.Price * float32(position.Count)
}

func (b *Bot) stopLoss(position *request.Position, stock *protocol.Stock) bool {
	return stock.Price <= position.StopLoss
}

func (b *Bot) takeProfit(position *request.Position, stock *protocol.Stock) bool {
	return stock.Price >= position.TakeProfit
}
