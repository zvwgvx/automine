package c2

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/automine/server/pkg/protocol"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
    // Allow Origin Check (For dev/testing, strictly allow trusted domains in prod)
    CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub
	// The websocket connection.
	Conn *websocket.Conn
	// Buffered channel of outbound messages.
	Send chan []byte
	// Agent Identity
	AgentID string
    Authenticated bool
    PubKey ed25519.PublicKey
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
        
        c.handleMessage(message)
	}
}

func (c *Client) handleMessage(msg []byte) {
    var envelope protocol.Envelope
    if err := json.Unmarshal(msg, &envelope); err != nil {
        log.Printf("[CLIENT] Invalid JSON: %v", err)
        return
    }

    // 1. Verify Signature
    // If Auth not yet established, PubKey MUST be present in Envelope.
    // If Auth established, we can use stored PubKey.
    
    var pubKey ed25519.PublicKey
    var err error
    
    if c.Authenticated && len(c.PubKey) > 0 {
        pubKey = c.PubKey
    } else {
        if envelope.PubKey == "" {
             log.Printf("[CLIENT] Missing Public Key for unauthenticated client")
             return
        }
        pubKeyBytes, err := hex.DecodeString(envelope.PubKey)
        if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
             log.Printf("[CLIENT] Invalid Public Key format")
             return
        }
        pubKey = ed25519.PublicKey(pubKeyBytes)
    }

    sigBytes, err := hex.DecodeString(envelope.Signature)
    if err != nil {
        log.Printf("[CLIENT] Invalid Signature format")
        return
    }

    // Verify: Sig(Payload) == Signature
    if !ed25519.Verify(pubKey, []byte(envelope.Payload), sigBytes) {
        log.Printf("[CLIENT] Signature Verification FAILED")
        return
    }

    // 2. Process Payload based on Type
    switch envelope.Type {
    case "auth":
        // Payload is JSON(AuthMessage)
        var auth protocol.AuthMessage
        // In real secure implementation, Payload is Base64(Encrypted), here implementation assumption:
        // For Phase 1, we assume Payload is just the JSON string (signed). 
        // If encrypted, we decrypt first.
        if err := json.Unmarshal([]byte(envelope.Payload), &auth); err != nil {
             log.Printf("[CLIENT] Invalid Auth Payload")
             return
        }
        
        c.AgentID = auth.AgentID
        c.PubKey = pubKey
        c.Authenticated = true
        
        // Register to Hub NOW (or update if already registered)
        c.Hub.Register <- c 
        log.Printf("[CLIENT] Authenticated: %s", c.AgentID)

    case "heartbeat":
        if !c.Authenticated { return }
        
        var hb protocol.HeartbeatMessage
        if err := json.Unmarshal([]byte(envelope.Payload), &hb); err != nil {
             return
        }
        // Update Metadata in Hub/Map (Not implemented in Hub struct in detail yet, but acceptable for now)
        log.Printf("[HEARTBEAT] %s | Mesh: %.2f | Wallet: ...", hb.AgentID, hb.MeshHealth)
        
    case "response":
        // Handle command responses
    }
}


// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}
    // Don't register to Hub yet. Wait for Auth message.
    
	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
