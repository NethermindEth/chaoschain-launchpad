package p2p

import "log"

// NodeWrapper wraps a Node instance to provide backward-compatible methods
// and help transition from the old p2p implementation to the new one
type NodeWrapper struct {
	*Node
	agentName string // Name of the agent using this wrapper (for logging/debugging)
}

// WrapNode creates a new NodeWrapper around an existing Node
func WrapNode(node *Node, agentName string) *NodeWrapper {
	return &NodeWrapper{
		Node:      node,
		agentName: agentName,
	}
}

// BroadcastMessage is a compatibility wrapper for the old BroadcastMessage method
// It ensures messages have all required fields before broadcasting
func (w *NodeWrapper) BroadcastMessage(msg Message) {
	// Ensure the message has an ID
	if msg.ID == "" {
		msg.ID = MessageID(GenerateUUID())
		log.Printf("NodeWrapper: Added missing ID to message for agent %s", w.agentName)
	}

	// Ensure the message has a timestamp
	if msg.Timestamp.IsZero() {
		msg.Timestamp = TimeNow()
		log.Printf("NodeWrapper: Added missing timestamp to message for agent %s", w.agentName)
	}

	// Ensure the sender ID is set
	if msg.SenderID == "" {
		msg.SenderID = w.AgentID
		log.Printf("NodeWrapper: Added missing sender ID (%s) to message for agent %s",
			w.AgentID, w.agentName)
	}

	// Use the enhanced BroadcastMessage function
	w.Node.BroadcastMessage(msg)
}

// SendDirectMessageByName sends a message directly to an agent by its name
// This is a convenience method for code that doesn't know agent IDs yet
func (w *NodeWrapper) SendDirectMessageByName(recipientName string, msgType string, data interface{}) error {
	// In a real implementation, you'd look up the AgentID by name from a registry
	// For demo purposes, we'll create an AgentID based on the name
	recipientID := AgentID(recipientName)

	// Create and send the message
	return w.SendDirectMessage(recipientID, msgType, data)
}

// CreateMessage is a helper to create a Message with the wrapper's agent ID as sender
func (w *NodeWrapper) CreateMessage(msgType string, data interface{}) Message {
	msg := NewMessage(msgType, data)
	msg.SenderID = w.AgentID
	return msg
}

// GetAgentName returns the name of the agent using this wrapper
func (w *NodeWrapper) GetAgentName() string {
	return w.agentName
}
