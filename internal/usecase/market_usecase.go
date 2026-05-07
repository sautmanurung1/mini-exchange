package usecase

import (
	"mini-exchange/internal/domain"
)

type MarketUseCase struct {
	marketRepo domain.MarketRepository
	tradeRepo  domain.TradeRepository
}

func NewMarketUseCase(marketRepo domain.MarketRepository, tradeRepo domain.TradeRepository) *MarketUseCase {
	return &MarketUseCase{
		marketRepo: marketRepo,
		tradeRepo:  tradeRepo,
	}
}

func (u *MarketUseCase) GetTicker(stockCode string) (*domain.Ticker, error) {
	return u.marketRepo.GetTicker(stockCode)
}

func (u *MarketUseCase) GetOrderBook(stockCode string) (*domain.OrderBook, error) {
	return u.marketRepo.GetOrderBook(stockCode)
}

func (u *MarketUseCase) GetTradeHistory(stockCode string) ([]*domain.Trade, error) {
	if stockCode == "" {
		return u.tradeRepo.GetAll()
	}
	return u.tradeRepo.GetByStock(stockCode)
}
