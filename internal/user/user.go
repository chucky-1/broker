// Package user handles each user
package user

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/chucky-1/broker/internal/repository"
	"github.com/chucky-1/broker/internal/request"
	log "github.com/sirupsen/logrus"

	"context"
	"sync"
)

// User keeps state each user
type User struct {
	muRep     *sync.Mutex
	rep       *repository.Repository
	id        int32
	muBalance sync.RWMutex
	balance   float32
	chPrice   chan *model.Price
	muPos     sync.RWMutex
	positions map[int32]map[int32]*model.Position // map[symbolID]map[position.ID]*position
}

// NewUser is constructor
func NewUser(ctx context.Context, id int32, balance float32, positions map[int32]map[int32]*model.Position,
	muRep *sync.Mutex, rep *repository.Repository) (*User, error) {
	u := User{
		muRep:     muRep,
		rep:       rep,
		id:        id,
		balance:   balance,
		chPrice:   make(chan *model.Price),
		positions: positions,
	}
	go func(ctx context.Context) {
		for {
			select {
			case <- ctx.Done():
				return
			case price := <- u.chPrice:
				for _, position := range u.positions[price.ID] {
					p := pnl(position, price)
					log.Infof("pnl for position %d is %f", position.ID, p)
					if stopLoss(position, price) {
						err := u.ClosePositionWithoutService(ctx, position, price)
						if err != nil {
							log.Error(err)
						}
					}
					if takeProfit(position, price) {
						err := u.ClosePositionWithoutService(ctx, position, price)
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

// ClosePositionWithoutService closes a position in the database
func (u *User) ClosePositionWithoutService(ctx context.Context, position *model.Position, p *model.Price) error {
	var price float32
	if position.IsBuy == true {
		price = p.Ask
	} else {
		price = p.Bid
	}
	sum := price * float32(position.Count)
	u.muRep.Lock()
	err := u.rep.ChangeBalance(ctx, u.id, sum)
	u.muRep.Unlock()
	if err != nil {
		return err
	}
	u.ChangeBalance(sum)
	u.muRep.Lock()
	err = u.rep.ClosePosition(ctx, &request.ClosePosition{
		ID:         position.ID,
		PriceClose: price,
	})
	u.muRep.Unlock()
	if err != nil {
		u.ChangeBalance(-sum)
		u.muRep.Lock()
		err = u.rep.ChangeBalance(ctx, u.id, -sum)
		u.muRep.Unlock()
		if err != nil {
			log.Error(err)
		}
		return err
	}

	u.muPos.Lock()
	delete(u.positions[position.SymbolID], position.ID)
	u.muPos.Unlock()
	return nil
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

// pnl is Profit and loss. Shows how much you earned or lost
func pnl(position *model.Position, price *model.Price) float32 {
	return price.Bid * float32(position.Count) - position.PriceOpen * float32(position.Count)
}

func stopLoss(position *model.Position, price *model.Price) bool {
	if position.IsBuy {
		return price.Ask <= position.StopLoss
	}
	return price.Bid >= position.StopLoss
}

func takeProfit(position *model.Position, price *model.Price) bool {
	if position.IsBuy {
		return price.Ask >= position.TakeProfit
	}
	return price.Bid <= position.TakeProfit
}

// GetID returns id
func (u *User) GetID() int32 {
	return u.id
}

// GetChanPrice returns chan of price
func (u *User) GetChanPrice() chan *model.Price {
	return u.chPrice
}

// GetPositions returns positions
func (u *User) GetPositions() (map[int32]map[int32]*model.Position, *sync.RWMutex) {
	return u.positions, &u.muPos
}
