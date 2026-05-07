package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"mini-exchange/internal/domain"
)

type OrderUseCase struct {
	orderRepo      domain.OrderRepository
	marketRepo     domain.MarketRepository
	matchingEngine domain.MatchingEngine
}

func NewOrderUseCase(
	orderRepo domain.OrderRepository,
	marketRepo domain.MarketRepository,
	matchingEngine domain.MatchingEngine,
) *OrderUseCase {
	return &OrderUseCase{
		orderRepo:      orderRepo,
		marketRepo:     marketRepo,
		matchingEngine: matchingEngine,
	}
}

func (u *OrderUseCase) CreateOrder(userID, stockCode string, side domain.OrderSide, price float64, quantity int64) (*domain.Order, error) {
	order := &domain.Order{
		ID:        uuid.New().String(),
		UserID:    userID,
		StockCode: stockCode,
		Side:      side,
		Price:     price,
		Quantity:  quantity,
		Remaining: quantity,
		Status:    domain.StatusOpen,
		CreatedAt: time.Now(),
	}

	if err := u.orderRepo.Create(order); err != nil {
		return nil, err
	}

	trades, err := u.matchingEngine.Match(order)
	if err != nil {
		return nil, err
	}

	// Publish updates to Redis
	ctx := context.Background()
	u.marketRepo.Publish(ctx, "events", domain.Event{
		Type:      domain.EventOrder,
		StockCode: order.StockCode,
		Data:      order,
	})
	for _, t := range trades {
		u.marketRepo.Publish(ctx, "events", domain.Event{
			Type:      domain.EventTrade,
			StockCode: t.StockCode,
			Data:      t,
		})
	}

	return order, nil
}

func (u *OrderUseCase) GetOrders(stockCode string, status domain.OrderStatus) ([]*domain.Order, error) {
	return u.orderRepo.GetAll(stockCode, status)
}
