package repository

import (
	"github.com/chucky-1/broker/internal/model"
	"github.com/go-redis/cache/v8"
	"strconv"
	"time"

	"context"
)

type Cache struct {
	cache *cache.Cache
}

func NewCache(cache *cache.Cache) *Cache {
	return &Cache{cache: cache}
}

func (c *Cache) Set(stock *model.Stock) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	key := strconv.Itoa(int(stock.ID))
	err := c.cache.Set(&cache.Item{
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

func (c *Cache) Get(key string) (*model.Stock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var stock model.Stock
	err := c.cache.Get(ctx, key, &stock)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}
