// Package user handles each user
package user

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"

	"context"
	"sync"
)

// User keeps state each user
type User struct {
	id        int32
	muBalance sync.RWMutex
	balance   float32
	chPrice   chan *model.Price
	positions *sync.Map // map[symbolID]map[position.ID]*position
	closer    request.PositionCloser
}

// NewUser is constructor
func NewUser(ctx context.Context, id int32, balance float32, positions *sync.Map, closer request.PositionCloser) (*User, error) {
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
			case <-ctx.Done():
				return
			case price := <-u.chPrice:
				p, ok := u.positions.Load(price.ID)
				if !ok {
					continue
				}
				pos := p.(map[int32]*model.Position)
				for _, position := range pos{
					position.BidClose = price.Bid
					position.AskClose = price.Ask
					pn := pnl(position)
					log.Infof("pnl for position %d is %f", position.ID, pn)
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
			}
		}
	}(ctx)
	return &u, nil
}

// OpenPosition appends position
func (u *User) OpenPosition(position *model.Position) {
	allPositions, ok := u.positions.Load(position.SymbolID)
	if !ok {
		m := make(map[int32]*model.Position)
		m[position.ID] = position
		u.positions.Store(position.SymbolID, m)
	} else {
		positions := allPositions.(map[int32]*model.Position)
		positions[position.ID] = position
	}
}

// ClosePosition delete position
func (u *User) ClosePosition(symbolID, positionID int32) {
	m, ok := u.positions.Load(symbolID)
	if !ok {
		return
	}
	positions := m.(map[int32]*model.Position)
	delete(positions, positionID)
	if len(positions) == 0 {
		u.positions.Delete(symbolID)
	}
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

	p, ok := u.positions.Load(position.SymbolID)
	if !ok {
		log.Errorf("position %d successful close but didn't delete from map", position.ID)
	}
	positions := p.(map[int32]*model.Position)
	delete(positions, position.ID)
	if len(positions) == 0 {
		u.positions.Delete(position.SymbolID)
	}
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

	u.positions.Range(func(key, value interface{}) bool {
		positions := value.(map[int32]*model.Position)
		for _, pos := range positions {
			p := pnl(pos)
			if position.IsBuy {
				bln = bln + pos.PriceOpen * float32(position.Count) + p
			} else {
				bln = bln - pos.PriceOpen * float32(position.Count) + p
			}
		}
		return true
	})
	return bln < 0
}

// GetID returns id
func (u *User) GetID() int32 {
	return u.id
}

// GetChanPrice returns chan of price
func (u *User) GetChanPrice() chan *model.Price {
	return u.chPrice
}
