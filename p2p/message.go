package p2p

import (
	"time"
)

// MessageID is a unique identifier for a message
type MessageID string

// AgentID represents a unique identifier for an agent
type AgentID string

// Message represents a P2P network message
type Message struct {
	ID          MessageID   `json:"id"`           // Unique message identifier
	Type        string      `json:"type"`         // Message type
	Data        interface{} `json:"data"`         // Message payload
	SenderID    AgentID     `json:"sender_id"`    // ID of the sender agent
	RecipientID AgentID     `json:"recipient_id"` // ID of the recipient agent (empty for broadcast)
	Timestamp   time.Time   `json:"timestamp"`    // Time when the message was created
	TTL         int         `json:"ttl"`          // Time-to-live for message hops (prevents infinite propagation)
	Signature   []byte      `json:"signature"`    // Optional digital signature for message authenticity
}

// NewMessage creates a new message with default values
func NewMessage(msgType string, data interface{}) Message {
	return Message{
		ID:        MessageID(GenerateUUID()), // Implement or import UUID generation
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		TTL:       10, // Default TTL value
	}
}

// SetSender sets the sender ID for the message
func (m *Message) SetSender(senderID AgentID) *Message {
	m.SenderID = senderID
	return m
}

// SetRecipient sets the recipient ID for the message (for direct messaging)
func (m *Message) SetRecipient(recipientID AgentID) *Message {
	m.RecipientID = recipientID
	return m
}

// IsDirected returns true if the message is directed to a specific recipient
func (m *Message) IsDirected() bool {
	return m.RecipientID != ""
}

// IsBroadcast returns true if the message is a broadcast message
func (m *Message) IsBroadcast() bool {
	return m.RecipientID == ""
}

// String returns a string representation of the message for logging
func (m Message) String() string {
	return string(m.ID)
}
