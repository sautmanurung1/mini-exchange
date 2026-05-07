package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	deliveryHttp "mini-exchange/internal/delivery/http"
	deliveryWs "mini-exchange/internal/delivery/ws"
	"mini-exchange/internal/domain"
	"mini-exchange/internal/repository/database"
	repoRedis "mini-exchange/internal/repository/redis"
	"mini-exchange/internal/usecase"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 1. Initialize SQLite (Using local file trading.db)
	db, err := gorm.Open(sqlite.Open("trading.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("SQLite connected successfully")

	// Auto Migration
	db.AutoMigrate(&domain.Order{}, &domain.Trade{}, &domain.User{})

	// 2. Initialize Redis
	redisAddr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	if os.Getenv("REDIS_HOST") == "" {
		redisAddr = "localhost:6379" // Default fallback
	}
	
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Market data caching will be disabled.", err)
	} else {
		log.Println("Redis connected successfully")
	}

	// Repositories
	orderRepo := database.NewOrderRepo(db)
	tradeRepo := database.NewTradeRepo(db)
	userRepo := database.NewUserRepo(db)
	marketRepo := repoRedis.NewMarketRepo(rdb)

	// Use Cases
	authUC := usecase.NewAuthUseCase(userRepo)
	matchingEngine := usecase.NewMatchingEngine(orderRepo, tradeRepo, marketRepo)
	orderUC := usecase.NewOrderUseCase(orderRepo, marketRepo, matchingEngine)
	marketUC := usecase.NewMarketUseCase(marketRepo, tradeRepo)

	// Delivery
	handler := deliveryHttp.NewHandler(orderUC, marketUC, authUC)
	middleware := deliveryHttp.NewMiddleware(authUC)
	hub := deliveryWs.NewHub(marketRepo)

	go hub.Run()

	// Redis Monitor (Menampilkan data Redis ke Terminal)
	redisMonitor := usecase.NewRedisMonitor(marketRepo)
	go redisMonitor.Start(context.Background())

	// Simulation
	simulator := usecase.NewMarketSimulator(orderUC, []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"})
	go simulator.Start()

	// Routes
	http.HandleFunc("/api/auth/register", handler.Register)
	http.HandleFunc("/api/auth/login", handler.Login)

	http.HandleFunc("/api/orders", middleware.RateLimit(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			middleware.Auth(handler.CreateOrder)(w, r)
		} else {
			handler.GetOrders(w, r)
		}
	}))
	http.HandleFunc("/api/trades", middleware.RateLimit(handler.GetTradeHistory))
	http.HandleFunc("/api/market/snapshot", middleware.RateLimit(handler.GetMarketSnapshot))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		deliveryWs.ServeWs(hub, w, r)
	})

	log.Printf("Trading Backend API live at http://localhost:%s", port)
	log.Println("Terminal Monitor active. Displaying Redis data...")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
