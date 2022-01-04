// Package user handles each user
package user

import (
	"context"
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"
	"sync"
)

// User keeps state each user
type User struct {
	id        int32
	muBalance sync.RWMutex
	balance   float32
	chPrice   chan *model.Price
	muPos     sync.RWMutex
	positions map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
	closer    request.PositionCloser
}

// NewUser is constructor
func NewUser(ctx context.Context, id int32, balance float32, positions map[int32]map[int32]*model.Position,
	closer request.PositionCloser) (*User, error) {
	u := User{
		id:        id,
		balance:   balance,
		chPrice:   make(chan *model.Price),
		positions: positions,
		closer:    closer,
	}
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- u.chPrice:
				u.muPos.RLock()
				for _, position := range u.positions[price.ID] {
					position.BidClose = price.Bid
					position.AskClose = price.Ask
					p := pnl(position)
					log.Infof("pnl for position %d is %f", position.ID, p)
					if stopLoss(position) {
						err := u.close(ctx, position)
						if err != nil {
							log.Error(err)
						}
					}
					if takeProfit(position) {
						err := u.close(ctx, position)
						if err != nil {
							log.Error(err)
						}
					}
					if u.marginCall(position) {
						err := u.close(ctx, position)
						if err != nil {
							log.Error(err)
						}
					}
				}
				u.muPos.RUnlock()
			}
		}
	}(ctx)
	return &u, nil
}

// OpenPosition appends position
func (u *User) OpenPosition(position *model.Position) {
	u.muPos.Lock()
	defer u.muPos.Unlock()
	allPositions, ok := u.positions[position.SymbolID]
	if !ok {
		u.positions[position.SymbolID] = make(map[int32]*model.Position)
		u.positions[position.SymbolID][position.ID] = position
	} else {
		allPositions[position.ID] = position
	}
}

// ClosePosition delete position
func (u *User) ClosePosition(symbolID, positionID int32) {
	u.muPos.Lock()
	delete(u.positions[symbolID], positionID)
	u.muPos.Unlock()
}

// GetBalance returns balance
func (u *User) GetBalance() float32 {
	u.muBalance.Lock()
	defer u.muBalance.Unlock()
	return u.balance
}

// ChangeBalance changes balance in user's struct
func (u *User) ChangeBalance(sum float32) {
	u.muBalance.Lock()
	u.balance += sum
	u.muBalance.Unlock()
}

func (u *User) close(ctx context.Context, position *model.Position) error {
	err := u.closer.Close(ctx, position)
	if err != nil {
		return err
	}
	if position.IsBuy {
		u.muBalance.Lock()
		u.balance += position.AskClose * float32(position.Count)
		u.muBalance.Unlock()
	} else {
		u.muBalance.Lock()
		u.balance -= position.BidClose * float32(position.Count)
		u.muBalance.Unlock()
	}
	delete(u.positions[position.SymbolID], position.ID)
	return nil
}

// pnl is Profit and loss. Shows how much you earned or lost
func pnl(position *model.Position) float32 {
	if position.IsBuy {
		return position.AskClose * float32(position.Count) - position.PriceOpen * float32(position.Count)
	}
	return position.PriceOpen * float32(position.Count) - position.BidClose * float32(position.Count)
}

func stopLoss(position *model.Position) bool {
	if position.IsBuy {
		return position.AskClose <= position.StopLoss
	}
	return position.BidClose >= position.StopLoss
}

func takeProfit(position *model.Position) bool {
	if position.IsBuy {
		return position.AskClose >= position.TakeProfit
	}
	return position.BidClose <= position.TakeProfit
}

func (u *User) marginCall(position *model.Position) bool {
	u.muBalance.RLock()
	bln := u.balance
	u.muBalance.RUnlock()

	for _, positions := range u.positions {
		for _, pos := range positions {
			p := pnl(pos)
			if position.IsBuy {
				bln = bln + pos.PriceOpen * float32(position.Count) + p
			} else {
				bln = bln - pos.PriceOpen * float32(position.Count) + p
			}
		}
	}
	if bln < 0 {
		return true
	}
	return false
}

// GetID returns id
func (u *User) GetID() int32 {
	return u.id
}

// GetChanPrice returns chan of price
func (u *User) GetChanPrice() chan *model.Price {
	return u.chPrice
}
