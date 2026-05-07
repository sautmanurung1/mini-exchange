package http

import (
	"encoding/json"
	"net/http"

	"mini-exchange/internal/domain"
)

type Handler struct {
	orderUC  domain.OrderUseCase
	marketUC domain.MarketUseCase
	authUC   domain.AuthUseCase
}

func NewHandler(orderUC domain.OrderUseCase, marketUC domain.MarketUseCase, authUC domain.AuthUseCase) *Handler {
	return &Handler{
		orderUC:  orderUC,
		marketUC: marketUC,
		authUC:   authUC,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.authUC.Register(r.Context(), req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.authUC.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		StockCode string           `json:"stock_code"`
		Side      domain.OrderSide `json:"side"`
		Price     float64          `json:"price"`
		Quantity  int64            `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	order, err := h.orderUC.CreateOrder(userID, req.StockCode, req.Side, req.Price, req.Quantity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	stockCode := r.URL.Query().Get("stock")
	status := domain.OrderStatus(r.URL.Query().Get("status"))

	orders, err := h.orderUC.GetOrders(stockCode, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *Handler) GetTradeHistory(w http.ResponseWriter, r *http.Request) {
	stockCode := r.URL.Query().Get("stock")
	trades, err := h.marketUC.GetTradeHistory(stockCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trades)
}

func (h *Handler) GetMarketSnapshot(w http.ResponseWriter, r *http.Request) {
	stockCode := r.URL.Query().Get("stock")
	if stockCode == "" {
		http.Error(w, "stock parameter is required", http.StatusBadRequest)
		return
	}

	ticker, _ := h.marketUC.GetTicker(stockCode)
	orderBook, _ := h.marketUC.GetOrderBook(stockCode)
	trades, _ := h.marketUC.GetTradeHistory(stockCode)

	snapshot := struct {
		Ticker    *domain.Ticker    `json:"ticker"`
		OrderBook *domain.OrderBook `json:"order_book"`
		Trades    []*domain.Trade   `json:"trades"`
	}{
		Ticker:    ticker,
		OrderBook: orderBook,
		Trades:    trades,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}
