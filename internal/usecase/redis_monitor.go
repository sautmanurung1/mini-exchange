package usecase

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"mini-exchange/internal/domain"
)

type RedisMonitor struct {
	marketRepo domain.MarketRepository
}

func NewRedisMonitor(marketRepo domain.MarketRepository) *RedisMonitor {
	return &RedisMonitor{marketRepo: marketRepo}
}

func (m *RedisMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	
	for {
		select {
		case <-ticker.C:
			m.clearTerminal()
			
			data, err := m.marketRepo.GetAllTickers(ctx)
			if err != nil {
				fmt.Printf("❌ [REDIS ERROR]: %v\n", err)
				continue
			}

			fmt.Println("================================================================")
			fmt.Println("🚀 REALTIME TRADING TERMINAL MONITOR")
			fmt.Printf("⏰ Time: %s | Status: ONLINE\n", time.Now().Format("15:04:05"))
			fmt.Println("================================================================")
			fmt.Printf("%-12s | %-12s | %-10s | %-10s\n", "STOCK", "PRICE", "CHANGE", "VOLUME")
			fmt.Println("----------------------------------------------------------------")

			if len(data) == 0 {
				fmt.Println("Waiting for market data...")
			} else {
				// Sort keys for consistent display
				keys := make([]string, 0, len(data))
				for k := range data {
					keys = append(keys, k)
				}
				sort.Strings(keys)

				for _, key := range keys {
					val := data[key]
					// Simple display of the raw JSON or we could parse it
					// For beauty, let's just print the key and value for now
					// In a real terminal UI we'd parse the JSON
					fmt.Printf("%-12s | %s\n", key, val)
				}
			}
			fmt.Println("================================================================")
			fmt.Println("Daftar data di atas diambil langsung dari Redis secara realtime.")
			fmt.Println("Tekan Ctrl+C untuk keluar.")

		case <-ctx.Done():
			return
		}
	}
}

func (m *RedisMonitor) clearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}
