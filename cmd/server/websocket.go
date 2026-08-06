package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

func NewWSHub() *WSHub {
	return &WSHub{clients: make(map[*websocket.Conn]bool)}
}

func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade: %v", err)
		return
	}
	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()
	log.Printf("WS client connected (%d total)", len(h.clients))

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Printf("WS client disconnected (%d remaining)", len(h.clients))
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func (h *WSHub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
}
