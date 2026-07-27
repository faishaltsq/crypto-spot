package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/crypto-spot-signal/internal/domain"
	"github.com/gorilla/websocket"
)

type client struct {
	conn         *websocket.Conn
	send         chan []byte
	mu           sync.RWMutex
	comparePairs map[string]struct{}
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*client]struct{}
	upgrader websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{
		conn:         conn,
		send:         make(chan []byte, 32),
		comparePairs: make(map[string]struct{}),
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	go h.writer(c)
	h.reader(c)
}

func (h *Hub) reader(c *client) {
	defer h.remove(c)
	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		h.handleSubscription(c, raw)
	}
}

func (h *Hub) handleSubscription(c *client, raw []byte) {
	var message struct {
		Action  string   `json:"action"`
		Channel string   `json:"channel"`
		Pairs   []string `json:"pairs"`
	}
	if err := json.Unmarshal(raw, &message); err != nil || message.Channel != "compare" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, pair := range message.Pairs {
		pair = strings.ToUpper(strings.TrimSpace(pair))
		if pair == "" {
			continue
		}
		if message.Action == "subscribe" {
			c.comparePairs[pair] = struct{}{}
		}
		if message.Action == "unsubscribe" {
			delete(c.comparePairs, pair)
		}
	}
}

func (h *Hub) writer(c *client) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

func (h *Hub) Broadcast(event string, data interface{}) {
	payload, err := json.Marshal(domain.WSMessage{
		Event:     event,
		Timestamp: time.Now(),
		Data:      data,
	})
	if err != nil {
		log.Printf("websocket marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			go h.remove(c)
		}
	}
}

func (h *Hub) BroadcastCompare(event, symbol string, data interface{}) {
	payload, err := json.Marshal(domain.WSMessage{Event: event, Timestamp: time.Now(), Data: data})
	if err != nil {
		return
	}
	symbol = strings.ToUpper(symbol)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.mu.RLock()
		_, subscribed := c.comparePairs[symbol]
		c.mu.RUnlock()
		if !subscribed {
			continue
		}
		select {
		case c.send <- payload:
		default:
			go h.remove(c)
		}
	}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if _, exists := h.clients[c]; exists {
		delete(h.clients, c)
		close(c.send)
		_ = c.conn.Close()
	}
	h.mu.Unlock()
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
