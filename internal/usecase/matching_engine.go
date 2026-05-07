package usecase

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"mini-exchange/internal/domain"
)

type MatchingEngine struct {
	orderRepo  domain.OrderRepository
	tradeRepo  domain.TradeRepository
	marketRepo domain.MarketRepository
	muMap      sync.Map // Sharded lock for matching to prevent race conditions during matching per stock
}

func NewMatchingEngine(
	orderRepo domain.OrderRepository,
	tradeRepo domain.TradeRepository,
	marketRepo domain.MarketRepository,
) *MatchingEngine {
	return &MatchingEngine{
		orderRepo:  orderRepo,
		tradeRepo:  tradeRepo,
		marketRepo: marketRepo,
	}
}

func (e *MatchingEngine) getLock(stockCode string) *sync.Mutex {
	m, _ := e.muMap.LoadOrStore(stockCode, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func (e *MatchingEngine) Match(newOrder *domain.Order) ([]*domain.Trade, error) {
	lock := e.getLock(newOrder.StockCode)
	lock.Lock()
	defer lock.Unlock()

	var trades []*domain.Trade

	// Database transaction for atomicity
	err := e.orderRepo.ExecuteTx(func(txOrderRepo domain.OrderRepository, txTradeRepo domain.TradeRepository) error {
		var counterSide domain.OrderSide
		if newOrder.Side == domain.SideBuy {
			counterSide = domain.SideSell
		} else {
			counterSide = domain.SideBuy
		}

		counterOrders, err := txOrderRepo.GetOpenOrders(newOrder.StockCode, counterSide)
		if err != nil {
			return err
		}

		// Sort counter orders by price and time (FIFO)
		sort.Slice(counterOrders, func(i, j int) bool {
			if newOrder.Side == domain.SideBuy {
				if counterOrders[i].Price == counterOrders[j].Price {
					return counterOrders[i].CreatedAt.Before(counterOrders[j].CreatedAt)
				}
				return counterOrders[i].Price < counterOrders[j].Price
			} else {
				if counterOrders[i].Price == counterOrders[j].Price {
					return counterOrders[i].CreatedAt.Before(counterOrders[j].CreatedAt)
				}
				return counterOrders[i].Price > counterOrders[j].Price
			}
		})

		for _, counterOrder := range counterOrders {
			if newOrder.Remaining <= 0 {
				break
			}

			match := false
			if newOrder.Side == domain.SideBuy && newOrder.Price >= counterOrder.Price {
				match = true
			} else if newOrder.Side == domain.SideSell && newOrder.Price <= counterOrder.Price {
				match = true
			}

			if match {
				matchQty := newOrder.Remaining
				if counterOrder.Remaining < matchQty {
					matchQty = counterOrder.Remaining
				}

				matchPrice := counterOrder.Price // Price of the existing order

				trade := &domain.Trade{
					ID:          uuid.New().String(),
					StockCode:   newOrder.StockCode,
					Price:       matchPrice,
					Quantity:    matchQty,
					CreatedAt:   time.Now(),
				}

				if newOrder.Side == domain.SideBuy {
					trade.BuyerID = newOrder.UserID
					trade.SellerID = counterOrder.UserID
					trade.BuyOrderID = newOrder.ID
					trade.SellOrderID = counterOrder.ID
				} else {
					trade.BuyerID = counterOrder.UserID
					trade.SellerID = newOrder.UserID
					trade.BuyOrderID = counterOrder.ID
					trade.SellOrderID = newOrder.ID
				}

				trades = append(trades, trade)
				if err := txTradeRepo.Create(trade); err != nil {
					return err
				}

				newOrder.Remaining -= matchQty
				counterOrder.Remaining -= matchQty

				if counterOrder.Remaining == 0 {
					counterOrder.Status = domain.StatusFilled
				} else {
					counterOrder.Status = domain.StatusPartial
				}
				if err := txOrderRepo.Update(counterOrder); err != nil {
					return err
				}

				if newOrder.Remaining == 0 {
					newOrder.Status = domain.StatusFilled
				} else {
					newOrder.Status = domain.StatusPartial
				}
				if err := txOrderRepo.Update(newOrder); err != nil {
					return err
				}

				// Update Ticker in Redis (outside DB Tx but while lock is held to keep order)
				ticker, _ := e.marketRepo.GetTicker(newOrder.StockCode)
				ticker.LastPrice = matchPrice
				ticker.Volume += matchQty
				
				// Optional logic: we update Redis ticker immediately because it does not have Tx support
				e.marketRepo.UpdateTicker(ticker)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Update OrderBook in Redis
	e.updateOrderBook(newOrder.StockCode)

	return trades, nil
}

func (e *MatchingEngine) updateOrderBook(stockCode string) {
	openBuys, _ := e.orderRepo.GetOpenOrders(stockCode, domain.SideBuy)
	openSells, _ := e.orderRepo.GetOpenOrders(stockCode, domain.SideSell)

	ob := &domain.OrderBook{
		StockCode: stockCode,
		Bids:      aggregateLevels(openBuys, true),
		Offers:    aggregateLevels(openSells, false),
	}

	e.marketRepo.UpdateOrderBook(ob)
	e.marketRepo.Publish(context.Background(), "events", domain.Event{
		Type:      domain.EventOrderBook,
		StockCode: ob.StockCode,
		Data:      ob,
	})
}

func aggregateLevels(orders []*domain.Order, desc bool) []domain.OrderBookLevel {
	levelsMap := make(map[float64]int64)
	for _, o := range orders {
		levelsMap[o.Price] += o.Remaining
	}

	var prices []float64
	for p := range levelsMap {
		prices = append(prices, p)
	}

	sort.Slice(prices, func(i, j int) bool {
		if desc {
			return prices[i] > prices[j] // Descending for Bids
		}
		return prices[i] < prices[j] // Ascending for Offers
	})

	var levels []domain.OrderBookLevel
	for _, p := range prices {
		levels = append(levels, domain.OrderBookLevel{
			Price:    p,
			Quantity: levelsMap[p],
		})
	}
	return levels
}
