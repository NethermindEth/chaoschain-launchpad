package p2p

// Message represents a P2P network message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
