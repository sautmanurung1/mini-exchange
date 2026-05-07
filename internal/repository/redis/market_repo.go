package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"mini-exchange/internal/domain"
)

type MarketRepo struct {
	client *redis.Client
	ctx    context.Context
}

func NewMarketRepo(client *redis.Client) *MarketRepo {
	return &MarketRepo{
		client: client,
		ctx:    context.Background(),
	}
}

func (r *MarketRepo) UpdateTicker(ticker *domain.Ticker) error {
	data, err := json.Marshal(ticker)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("ticker:%s", ticker.StockCode)
	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

func (r *MarketRepo) GetTicker(stockCode string) (*domain.Ticker, error) {
	key := fmt.Sprintf("ticker:%s", stockCode)
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return &domain.Ticker{StockCode: stockCode}, nil
	}
	var ticker domain.Ticker
	if err := json.Unmarshal([]byte(val), &ticker); err != nil {
		return &domain.Ticker{StockCode: stockCode}, nil
	}
	return &ticker, nil
}

func (r *MarketRepo) UpdateOrderBook(ob *domain.OrderBook) error {
	data, err := json.Marshal(ob)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("orderbook:%s", ob.StockCode)
	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

func (r *MarketRepo) GetOrderBook(stockCode string) (*domain.OrderBook, error) {
	key := fmt.Sprintf("orderbook:%s", stockCode)
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return &domain.OrderBook{StockCode: stockCode}, nil
	}
	var ob domain.OrderBook
	if err := json.Unmarshal([]byte(val), &ob); err != nil {
		return &domain.OrderBook{StockCode: stockCode}, nil
	}
	return &ob, nil
}

func (r *MarketRepo) GetAllTickers(ctx context.Context) (map[string]string, error) {
	keys, err := r.client.Keys(ctx, "ticker:*").Result()
	if err != nil {
		return nil, err
	}
	
	results := make(map[string]string)
	for _, key := range keys {
		val, _ := r.client.Get(ctx, key).Result()
		results[key] = val
	}
	return results, nil
}

func (r *MarketRepo) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, channel, data).Err()
}

func (r *MarketRepo) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	pubsub := r.client.Subscribe(ctx, channel)
	ch := make(chan string)
	go func() {
		defer pubsub.Close()
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()
	return ch, nil
}
