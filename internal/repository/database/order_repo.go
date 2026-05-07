package database

import (
	"mini-exchange/internal/domain"
	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) Create(order *domain.Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepo) GetByID(id string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepo) GetAll(stockCode string, status domain.OrderStatus) ([]*domain.Order, error) {
	var orders []*domain.Order
	query := r.db.Model(&domain.Order{})
	if stockCode != "" {
		query = query.Where("stock_code = ?", stockCode)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *OrderRepo) Update(order *domain.Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepo) GetOpenOrders(stockCode string, side domain.OrderSide) ([]*domain.Order, error) {
	var orders []*domain.Order
	err := r.db.Where("stock_code = ? AND side = ? AND status IN (?, ?)", 
		stockCode, side, domain.StatusOpen, domain.StatusPartial).
		Find(&orders).Error
	return orders, err
}

func (r *OrderRepo) ExecuteTx(fn func(domain.OrderRepository, domain.TradeRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txOrderRepo := NewOrderRepo(tx)
		txTradeRepo := NewTradeRepo(tx)
		return fn(txOrderRepo, txTradeRepo)
	})
}
