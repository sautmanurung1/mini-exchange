package domain

import "time"

type Trade struct {
	ID         string    `json:"id"`
	StockCode  string    `json:"stock_code"`
	Price      float64   `json:"price"`
	Quantity   int64     `json:"quantity"`
	BuyerID    string    `json:"buyer_id"`
	SellerID   string    `json:"seller_id"`
	BuyOrderID string    `json:"buy_order_id"`
	SellOrderID string   `json:"sell_order_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type TradeRepository interface {
	Create(trade *Trade) error
	GetByStock(stockCode string) ([]*Trade, error)
	GetAll() ([]*Trade, error)
}
