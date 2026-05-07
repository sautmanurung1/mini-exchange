package usecase_test

import (
	"testing"

	"mini-exchange/internal/domain"
	"mini-exchange/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMatchingEngine struct {
	mock.Mock
}

func (m *MockMatchingEngine) Match(newOrder *domain.Order) ([]*domain.Trade, error) {
	args := m.Called(newOrder)
	return args.Get(0).([]*domain.Trade), args.Error(1)
}

func TestOrderUseCase_CreateOrder(t *testing.T) {
	orderRepo := new(MockOrderRepo)
	marketRepo := new(MockMarketRepo)
	matchingEngine := new(MockMatchingEngine)
	uc := usecase.NewOrderUseCase(orderRepo, marketRepo, matchingEngine)

	orderRepo.On("Create", mock.Anything).Return(nil)
	marketRepo.On("Publish", mock.Anything, "events", mock.Anything).Return(nil)
	matchingEngine.On("Match", mock.Anything).Return([]*domain.Trade{}, nil)

	order, err := uc.CreateOrder("test-user", "AAPL", domain.SideBuy, 150.0, 100)
	
	assert.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, "AAPL", order.StockCode)
	
	orderRepo.AssertExpectations(t)
	marketRepo.AssertExpectations(t)
	matchingEngine.AssertExpectations(t)
}
