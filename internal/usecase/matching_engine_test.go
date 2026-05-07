package usecase_test

import (
	"context"
	"testing"
	"time"

	"mini-exchange/internal/domain"
	"mini-exchange/internal/usecase"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOrderRepo struct {
	mock.Mock
}

func (m *MockOrderRepo) Create(order *domain.Order) error {
	args := m.Called(order)
	return args.Error(0)
}
func (m *MockOrderRepo) GetByID(id string) (*domain.Order, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Order), args.Error(1)
}
func (m *MockOrderRepo) GetAll(stockCode string, status domain.OrderStatus) ([]*domain.Order, error) {
	return nil, nil
}
func (m *MockOrderRepo) Update(order *domain.Order) error { return nil }
func (m *MockOrderRepo) GetOpenOrders(stockCode string, side domain.OrderSide) ([]*domain.Order, error) {
	args := m.Called(stockCode, side)
	return args.Get(0).([]*domain.Order), args.Error(1)
}
func (m *MockOrderRepo) ExecuteTx(fn func(domain.OrderRepository, domain.TradeRepository) error) error {
	return fn(m, new(MockTradeRepo))
}

type MockTradeRepo struct{ mock.Mock }

func (m *MockTradeRepo) Create(trade *domain.Trade) error                     { return nil }
func (m *MockTradeRepo) GetByStock(stockCode string) ([]*domain.Trade, error) { return nil, nil }
func (m *MockTradeRepo) GetAll() ([]*domain.Trade, error)                     { return nil, nil }

type MockMarketRepo struct{ mock.Mock }

func (m *MockMarketRepo) UpdateTicker(ticker *domain.Ticker) error   { return nil }
func (m *MockMarketRepo) UpdateOrderBook(ob *domain.OrderBook) error { return nil }
func (m *MockMarketRepo) GetTicker(stockCode string) (*domain.Ticker, error) {
	return &domain.Ticker{StockCode: stockCode}, nil
}
func (m *MockMarketRepo) GetOrderBook(stockCode string) (*domain.OrderBook, error) { return nil, nil }
func (m *MockMarketRepo) GetAllTickers(ctx context.Context) (map[string]string, error) {
	return nil, nil
}
func (m *MockMarketRepo) Publish(ctx context.Context, channel string, message interface{}) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}
func (m *MockMarketRepo) Subscribe(ctx context.Context, channel string) (<-chan string, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(<-chan string), args.Error(1)
}

func TestMatchingEngine_Match(t *testing.T) {
	orderRepo := new(MockOrderRepo)
	tradeRepo := new(MockTradeRepo)
	marketRepo := new(MockMarketRepo)
	engine := usecase.NewMatchingEngine(orderRepo, tradeRepo, marketRepo)
	sellOrder := &domain.Order{
		ID:        "sell-1",
		UserID:    "user-sell",
		StockCode: "AAPL",
		Side:      domain.SideSell,
		Price:     150.0,
		Quantity:  100,
		Remaining: 100,
		Status:    domain.StatusOpen,
		CreatedAt: time.Now(),
	}

	buyOrder := &domain.Order{
		ID:        "buy-1",
		UserID:    "user-buy",
		StockCode: "AAPL",
		Side:      domain.SideBuy,
		Price:     150.0,
		Quantity:  60,
		Remaining: 60,
		Status:    domain.StatusOpen,
		CreatedAt: time.Now(),
	}

	orderRepo.On("GetOpenOrders", "AAPL", domain.SideSell).Return([]*domain.Order{sellOrder}, nil)
	orderRepo.On("GetOpenOrders", "AAPL", domain.SideBuy).Return([]*domain.Order{buyOrder}, nil)
	orderRepo.On("GetByID", "buy-1").Return(&domain.Order{ID: "buy-1", Status: domain.StatusFilled, Remaining: 0}, nil)
	orderRepo.On("GetByID", "sell-1").Return(&domain.Order{ID: "sell-1", Status: domain.StatusPartial, Remaining: 40}, nil)
	marketRepo.On("UpdateOrderBook", mock.Anything).Return(nil)
	marketRepo.On("Publish", mock.Anything, "events", mock.Anything).Return(nil)
	trades, err := engine.Match(buyOrder)
	assert.NoError(t, err)
	assert.Len(t, trades, 1)
	assert.Equal(t, int64(60), trades[0].Quantity)

	updatedBuy, _ := orderRepo.GetByID("buy-1")
	assert.Equal(t, domain.StatusFilled, updatedBuy.Status)

	updatedSell, _ := orderRepo.GetByID("sell-1")
	assert.Equal(t, domain.StatusPartial, updatedSell.Status)
}
