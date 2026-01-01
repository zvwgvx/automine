package c2

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients map: AgentID -> Client
	Clients map[string]*Client

	// Inbound messages from clients
	Broadcast chan []byte

	// Register requests from the clients
	Register chan *Client

	// Unregister requests from clients
	Unregister chan *Client

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[string]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.AgentID] = client
			h.mu.Unlock()
			log.Printf("[HUB] Registered Agent: %s", client.AgentID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.AgentID]; ok {
				delete(h.Clients, client.AgentID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[HUB] Unregistered Agent: %s", client.AgentID)

		case message := <-h.Broadcast:
			// Broadcast logic (if needed), for now we do direct messaging mostly
            _ = message
		}
	}
}

// SendCommand sends a command to a specific agent
func (h *Hub) SendCommand(agentID string, cmd interface{}) bool {
	h.mu.RLock()
	client, ok := h.Clients[agentID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

    payloadBytes, _ := json.Marshal(cmd) // Plain for now, TODO: Encrypt
    
    // Wrap in Envelope (Server doesn't sign commands yet, but could)
    // For simplicity phase 1: Just send raw JSON or envelope
	client.Send <- payloadBytes
	return true
}

func (h *Hub) BroadcastCommand(cmd interface{}) int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    count := 0
    payloadBytes, _ := json.Marshal(cmd) // Serialize once

    for _, client := range h.Clients {
        select {
        case client.Send <- payloadBytes:
            count++
        default:
            close(client.Send)
            delete(h.Clients, client.AgentID)
        }
    }
    return count
}
