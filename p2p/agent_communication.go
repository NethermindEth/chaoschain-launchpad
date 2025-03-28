package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// AgentCommunicationAdapter connects the P2P layer with agent-specific communication
// protocols. It translates between different message formats and provides
// a standardized way for agents to communicate.
type AgentCommunicationAdapter struct {
	node          *Node
	agentID       string    // String representation of AgentID
	agentName     string    // Human-readable name
	agentType     string    // Type of agent (e.g., "validator", "producer")
	lastMessageID MessageID // Track the last message ID for deduplication
}

// AgentMessage defines a standard structure for agent-to-agent communication
type AgentMessage struct {
	ID             string                 `json:"id"`
	SenderID       string                 `json:"sender_id"`
	SenderName     string                 `json:"sender_name"`
	SenderType     string                 `json:"sender_type"`
	RecipientID    string                 `json:"recipient_id,omitempty"`
	RecipientType  string                 `json:"recipient_type,omitempty"` // Can target a type of agent
	Intent         string                 `json:"intent"`                   // e.g., "PROPOSAL", "VALIDATION", "DISCUSSION"
	ContentType    string                 `json:"content_type"`             // e.g., "TASK", "BLOCK", "VOTE"
	Content        interface{}            `json:"content"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"` // For threaded conversations
	InReplyTo      string                 `json:"in_reply_to,omitempty"`     // Message ID this is replying to
	Timestamp      time.Time              `json:"timestamp"`
	ExpirationTime time.Time              `json:"expiration_time,omitempty"` // Optional expiration
}

// NewAgentCommunicationAdapter creates a new adapter for agent communication
func NewAgentCommunicationAdapter(node *Node, agentName, agentType string) *AgentCommunicationAdapter {
	return &AgentCommunicationAdapter{
		node:      node,
		agentID:   string(node.AgentID),
		agentName: agentName,
		agentType: agentType,
	}
}

// SendDirectMessage sends a message directly to a specific agent
func (a *AgentCommunicationAdapter) SendDirectMessage(recipientID, intent, contentType string, content interface{}) error {
	agentMsg := AgentMessage{
		ID:          GenerateUUID(),
		SenderID:    a.agentID,
		SenderName:  a.agentName,
		SenderType:  a.agentType,
		RecipientID: recipientID,
		Intent:      intent,
		ContentType: contentType,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Convert agent message to P2P message
	p2pMsg := NewMessage("AGENT_MESSAGE", agentMsg)
	p2pMsg.SenderID = a.node.AgentID
	p2pMsg.RecipientID = AgentID(recipientID)

	// Send via node
	return a.node.SendDirectMessage(AgentID(recipientID), "AGENT_MESSAGE", agentMsg)
}

// BroadcastToType broadcasts a message to all agents of a specific type
func (a *AgentCommunicationAdapter) BroadcastToType(agentType, intent, contentType string, content interface{}) {
	agentMsg := AgentMessage{
		ID:            GenerateUUID(),
		SenderID:      a.agentID,
		SenderName:    a.agentName,
		SenderType:    a.agentType,
		RecipientType: agentType, // Target all agents of this type
		Intent:        intent,
		ContentType:   contentType,
		Content:       content,
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	// Convert agent message to P2P message
	p2pMsg := NewMessage("AGENT_TYPE_MESSAGE", agentMsg)
	p2pMsg.SenderID = a.node.AgentID

	// Broadcast to all peers
	a.node.BroadcastMessage(p2pMsg)
}

// BroadcastToAll broadcasts a message to all agents
func (a *AgentCommunicationAdapter) BroadcastToAll(intent, contentType string, content interface{}) {
	agentMsg := AgentMessage{
		ID:          GenerateUUID(),
		SenderID:    a.agentID,
		SenderName:  a.agentName,
		SenderType:  a.agentType,
		Intent:      intent,
		ContentType: contentType,
		Content:     content,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Convert agent message to P2P message
	p2pMsg := NewMessage("AGENT_BROADCAST", agentMsg)
	p2pMsg.SenderID = a.node.AgentID

	// Broadcast to all peers
	a.node.BroadcastMessage(p2pMsg)
}

// ReplyToMessage creates a reply to a specific message
func (a *AgentCommunicationAdapter) ReplyToMessage(originalMsg *AgentMessage, intent, contentType string, content interface{}) error {
	if originalMsg == nil {
		return fmt.Errorf("cannot reply to nil message")
	}

	// Create a reply message
	replyMsg := AgentMessage{
		ID:             GenerateUUID(),
		SenderID:       a.agentID,
		SenderName:     a.agentName,
		SenderType:     a.agentType,
		RecipientID:    originalMsg.SenderID,
		Intent:         intent,
		ContentType:    contentType,
		Content:        content,
		InReplyTo:      originalMsg.ID,
		ConversationID: originalMsg.ConversationID, // Maintain the conversation thread
		Timestamp:      time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	// If no conversation ID in original, use its ID as the conversation starter
	if replyMsg.ConversationID == "" {
		replyMsg.ConversationID = originalMsg.ID
	}

	// Send direct message to the original sender
	p2pMsg := NewMessage("AGENT_MESSAGE", replyMsg)
	p2pMsg.SenderID = a.node.AgentID
	p2pMsg.RecipientID = AgentID(originalMsg.SenderID)

	return a.node.SendDirectMessage(AgentID(originalMsg.SenderID), "AGENT_MESSAGE", replyMsg)
}

// Subscribe registers callbacks for different message types
func (a *AgentCommunicationAdapter) Subscribe(handler func(AgentMessage)) {
	// Handle direct messages
	a.node.Subscribe("AGENT_MESSAGE", func(data []byte) {
		var agentMsg AgentMessage
		if err := json.Unmarshal(data, &agentMsg); err != nil {
			log.Printf("Error parsing AGENT_MESSAGE: %v", err)
			return
		}

		// Check if this message is intended for this agent
		if agentMsg.RecipientID == a.agentID {
			handler(agentMsg)
		}
	})

	// Handle type-targeted messages
	a.node.Subscribe("AGENT_TYPE_MESSAGE", func(data []byte) {
		var agentMsg AgentMessage
		if err := json.Unmarshal(data, &agentMsg); err != nil {
			log.Printf("Error parsing AGENT_TYPE_MESSAGE: %v", err)
			return
		}

		// Check if this message is intended for this agent type
		if agentMsg.RecipientType == a.agentType {
			handler(agentMsg)
		}
	})

	// Handle broadcast messages
	a.node.Subscribe("AGENT_BROADCAST", func(data []byte) {
		var agentMsg AgentMessage
		if err := json.Unmarshal(data, &agentMsg); err != nil {
			log.Printf("Error parsing AGENT_BROADCAST: %v", err)
			return
		}

		// Process all broadcast messages
		handler(agentMsg)
	})
}

// AddConversationMetadata adds metadata to a message for conversation tracking
func (a *AgentCommunicationAdapter) AddConversationMetadata(msg *AgentMessage, key string, value interface{}) {
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[key] = value
}
