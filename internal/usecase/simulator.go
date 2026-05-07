package usecase

import (
	"math/rand"
	"time"

	"mini-exchange/internal/domain"
)

type MarketSimulator struct {
	orderUC domain.OrderUseCase
	symbols []string
}

func NewMarketSimulator(orderUC domain.OrderUseCase, symbols []string) *MarketSimulator {
	return &MarketSimulator{
		orderUC: orderUC,
		symbols: symbols,
	}
}

func (s *MarketSimulator) Start() {
	ticker := time.NewTicker(2 * time.Second)
	prices := make(map[string]float64)
	for _, sym := range s.symbols {
		prices[sym] = 100.0 + rand.Float64()*100.0
	}

	for range ticker.C {
		for _, sym := range s.symbols {
			// Randomly decide to place a BUY or SELL order
			side := domain.SideBuy
			if rand.Intn(2) == 0 {
				side = domain.SideSell
			}

			// Randomly fluctuate price
			change := (rand.Float64() * 2) - 1.0 // -1 to +1
			prices[sym] += change
			if prices[sym] < 1.0 {
				prices[sym] = 1.0
			}

			price := prices[sym]
			quantity := int64(rand.Intn(100) + 1) * 10

			// To increase matching probability, sometimes place orders at "market" prices
			// By slightly adjusting the price to overlap with opposite side
			matchChance := rand.Intn(10)
			if matchChance < 3 { // 30% chance to try to match
				if side == domain.SideBuy {
					price += 0.5
				} else {
					price -= 0.5
				}
			}

			go s.orderUC.CreateOrder("system-simulator", sym, side, price, quantity)
		}
	}
}
