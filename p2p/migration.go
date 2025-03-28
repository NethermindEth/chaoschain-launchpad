package p2p

import (
	"fmt"
	"log"
)

// MigrateNodesWithAgentIDs assigns unique agent IDs to all existing P2P nodes
// Should be called during application startup to ensure all nodes have proper identity
func MigrateNodesWithAgentIDs(nodes map[string]*Node) {
	log.Println("Migrating P2P nodes to use agent IDs...")

	for addr, node := range nodes {
		// Skip if the node already has an agent ID
		if node.AgentID != "" {
			continue
		}

		// Generate a unique ID from the node's address for consistency
		agentID := GenerateAgentIDFromAddress(addr)
		node.AgentID = agentID

		log.Printf("Assigned agent ID %s to node at %s", agentID, addr)
	}

	log.Println("Migration completed")
}

// GenerateAgentIDFromAddress creates a deterministic agent ID from an address
func GenerateAgentIDFromAddress(address string) AgentID {
	// To make it deterministic but unique, combine the address with a fixed prefix
	uniqueStr := fmt.Sprintf("agent:%s", address)
	return AgentID(uniqueStr)
}

// InitializeMessageTracking prepares the P2P node for message tracking
// This should be called for all nodes during startup to prevent memory leaks
func InitializeMessageTracking(node *Node) {
	if node.seenMessages == nil {
		node.seenMessages = make(map[MessageID]bool)
		log.Printf("Initialized message tracking for node at port %d", node.port)
	}
}

// PatchLegacyNode updates an existing node with the capabilities needed for the enhanced system
func PatchLegacyNode(node *Node) {
	// Ensure agent ID is set
	if node.AgentID == "" {
		node.AgentID = AgentID(GenerateUUID())
		log.Printf("Generated new agent ID %s for legacy node", node.AgentID)
	}

	// Ensure message tracking is initialized
	InitializeMessageTracking(node)

	// Ensure direct message subscribers map is initialized
	if node.directMsgSubs == nil {
		node.directMsgSubs = make(map[AgentID][]func(Message))
	}
}
