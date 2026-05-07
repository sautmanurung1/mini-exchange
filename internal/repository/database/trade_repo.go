package database

import (
	"mini-exchange/internal/domain"
	"gorm.io/gorm"
)

type TradeRepo struct {
	db *gorm.DB
}

func NewTradeRepo(db *gorm.DB) *TradeRepo {
	return &TradeRepo{db: db}
}

func (r *TradeRepo) Create(trade *domain.Trade) error {
	return r.db.Create(trade).Error
}

func (r *TradeRepo) GetByStock(stockCode string) ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.db.Where("stock_code = ?", stockCode).Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}

func (r *TradeRepo) GetAll() ([]*domain.Trade, error) {
	var trades []*domain.Trade
	if err := r.db.Find(&trades).Error; err != nil {
		return nil, err
	}
	return trades, nil
}
