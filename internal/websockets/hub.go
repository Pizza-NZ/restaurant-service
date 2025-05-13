package websockets

import (
	"sync"
)

type Hub struct {
	clients map[*Client]bool

	register chan *Client

	unregister chan *Client

	broadcast chan []byte

	stationChannels map[string]map[*Client]bool

	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:       make(chan []byte),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		stationChannels: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) RegisterStationClient(client *Client, stationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.stationChannels[stationID]; !ok {
		h.stationChannels[stationID] = make(map[*Client]bool)
	}
	h.stationChannels[stationID][client] = true
}

func (h *Hub) BroadcastToStation(stationID string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.stationChannels[stationID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
				delete(h.clients, client)
			}
		}
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				h.mu.Lock()
				for _, clients := range h.stationChannels {
					delete(clients, client)
				}
				h.mu.Unlock()
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
