package p2p

import "time"

// This file provides a compatibility layer for code that uses the old Message structure
// It allows legacy code to continue working while transitioning to the new enhanced Message format

// CreateMessage is a helper function to create a Message from just type and data
// This is mainly for backward compatibility with code that used the old Message structure
func CreateMessage(msgType string, data interface{}) Message {
	// Use the new NewMessage function but ensure we handle backward compatibility
	msg := NewMessage(msgType, data)
	return msg
}

// GetP2PNodeWithAgentID returns the default P2P node with an agent ID set to the provided value
// This helps with transitioning code that didn't previously need to specify agent identities
func GetP2PNodeWithAgentID(agentID string) *Node {
	node := GetP2PNode()
	// Only override if explicitly provided
	if agentID != "" {
		node.AgentID = AgentID(agentID)
	}
	return node
}

// LegacyBroadcastMessage is a compatibility wrapper for old code that directly created Message structs
// It ensures the message has an ID and other required fields before broadcasting
func (n *Node) LegacyBroadcastMessage(msg Message) {
	// Ensure the message has an ID
	if msg.ID == "" {
		msg.ID = MessageID(GenerateUUID())
	}

	// Ensure the message has a timestamp
	if msg.Timestamp.IsZero() {
		msg.Timestamp = TimeNow()
	}

	// Ensure the sender ID is set
	if msg.SenderID == "" {
		msg.SenderID = n.AgentID
	}

	// Use the enhanced BroadcastMessage function
	n.BroadcastMessage(msg)
}

// TimeNow returns the current time - makes testing easier by allowing mocking
func TimeNow() time.Time {
	return time.Now()
}
