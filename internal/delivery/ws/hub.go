package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"mini-exchange/internal/domain"
)

type ServerMessage struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

type ClientMessage struct {
	Action string `json:"action"`
	Symbol string `json:"symbol"`
}

type Hub struct {
	clients       map[*Client]bool
	subscriptions map[string]map[*Client]bool
	register      chan *Client
	unregister    chan *Client
	marketRepo    domain.MarketRepository
	mu            sync.RWMutex
}

func NewHub(marketRepo domain.MarketRepository) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		marketRepo:    marketRepo,
	}
}

func (h *Hub) Run() {
	ctx := context.Background()
	eventChan, err := h.marketRepo.Subscribe(ctx, "events")
	if err != nil {
		log.Fatalf("Failed to subscribe to Redis events: %v", err)
	}

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %p", client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Cleanup subscriptions
				for symbol, clients := range h.subscriptions {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.subscriptions, symbol)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %p", client)

		case payload := <-eventChan:
			h.handleRedisEvent(payload)
		}
	}
}

func (h *Hub) handleRedisEvent(payload string) {
	var event domain.Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	if event.StockCode == "" {
		return
	}

	h.broadcastToSubscribers(event.StockCode, string(event.Type), event.Data)
}

func (h *Hub) broadcastToSubscribers(symbol string, channel string, data interface{}) {
	msg := ServerMessage{
		Channel: channel,
		Data:    data,
	}
	payload, _ := json.Marshal(msg)

	h.mu.RLock()
	clients, ok := h.subscriptions[symbol]
	if !ok {
		h.mu.RUnlock()
		return
	}

	var targets []*Client
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		select {
		case client.send <- payload:
		default:
			// Non-blocking broadcast
		}
	}
}

func (h *Hub) Subscribe(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subscriptions[symbol] == nil {
		h.subscriptions[symbol] = make(map[*Client]bool)
	}
	h.subscriptions[symbol][client] = true
	log.Printf("Client %p subscribed to %s", client, symbol)
}

func (h *Hub) Unsubscribe(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.subscriptions[symbol]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subscriptions, symbol)
		}
	}
}
