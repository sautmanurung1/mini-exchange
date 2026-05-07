package domain

import "context"

type EventType string

const (
	EventTicker    EventType = "market.ticker"
	EventTrade     EventType = "market.trade"
	EventOrderBook EventType = "market.orderbook"
	EventOrder     EventType = "order.update"
)

type Event struct {
	Type      EventType   `json:"type"`
	StockCode string      `json:"stock_code"`
	Data      interface{} `json:"data"`
}

type Ticker struct {
	StockCode string  `json:"stock_code"`
	LastPrice float64 `json:"last_price"`
	Change    float64 `json:"change"`
	Volume    int64   `json:"volume"`
}

type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity int64   `json:"quantity"`
}

type OrderBook struct {
	StockCode string           `json:"stock_code"`
	Bids      []OrderBookLevel `json:"bids"`
	Offers    []OrderBookLevel `json:"offers"`
}

type MarketRepository interface {
	UpdateTicker(ticker *Ticker) error
	GetTicker(stockCode string) (*Ticker, error)
	UpdateOrderBook(ob *OrderBook) error
	GetOrderBook(stockCode string) (*OrderBook, error)
	GetAllTickers(ctx context.Context) (map[string]string, error)
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) (<-chan string, error)
}

type MarketUseCase interface {
	GetTicker(stockCode string) (*Ticker, error)
	GetOrderBook(stockCode string) (*OrderBook, error)
	GetTradeHistory(stockCode string) ([]*Trade, error)
}
