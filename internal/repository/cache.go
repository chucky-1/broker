package repository

import (
	"fmt"
	"github.com/chucky-1/broker/internal/model"
	"github.com/go-redis/cache/v8"
	"sync"

	"context"
	"strconv"
	"time"
)

// Cache stores the latest prices
type Cache interface {
	Set(stock *model.Stock) error
	Get(key string) (*model.Stock, error)
}

// LocalMap implements cache on go
type LocalMap struct {
	mu sync.RWMutex
	mp map[string]*model.Stock
}

// NewLocalMap is constructor
func NewLocalMap() *LocalMap {
	return &LocalMap{mp: make(map[string]*model.Stock)}
}

// Redis works with redis cache
type Redis struct {
	cache *cache.Cache
}

// NewRedis is constructor
func NewRedis(cache *cache.Cache) *Redis {
	return &Redis{cache: cache}
}

func (l *LocalMap) Set(stock *model.Stock) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := strconv.Itoa(int(stock.ID))
	l.mp[key] = stock
	return nil
}

func (l *LocalMap) Get(key string) (*model.Stock, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	stock, ok := l.mp[key]
	if !ok {
		return nil, fmt.Errorf("stock with id %s not found", key)
	}
	return stock, nil
}

// Set updates the cache
func (r *Redis) Set(stock *model.Stock) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	key := strconv.Itoa(int(stock.ID))
	err := r.cache.Set(&cache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: stock,
		TTL:   time.Hour * 24,
	})
	if err != nil {
		return err
	}
	return nil
}

// Get returns the meaning from the cache
func (r *Redis) Get(key string) (*model.Stock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var stock model.Stock
	err := r.cache.Get(ctx, key, &stock)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}
