package domain

import "time"

type OrderSide string
type OrderStatus string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"

	StatusOpen      OrderStatus = "OPEN"
	StatusPartial   OrderStatus = "PARTIAL_FILLED"
	StatusFilled    OrderStatus = "FILLED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	StockCode string      `json:"stock_code"`
	Side      OrderSide   `json:"side"`
	Price     float64     `json:"price"`
	Quantity  int64       `json:"quantity"`
	Remaining int64       `json:"remaining"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderRepository interface {
	Create(order *Order) error
	GetByID(id string) (*Order, error)
	GetAll(stockCode string, status OrderStatus) ([]*Order, error)
	Update(order *Order) error
	GetOpenOrders(stockCode string, side OrderSide) ([]*Order, error)
	ExecuteTx(fn func(OrderRepository, TradeRepository) error) error
}

type OrderUseCase interface {
	CreateOrder(userID, stockCode string, side OrderSide, price float64, quantity int64) (*Order, error)
	GetOrders(stockCode string, status OrderStatus) ([]*Order, error)
}

type MatchingEngine interface {
	Match(newOrder *Order) ([]*Trade, error)
}
