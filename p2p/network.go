package p2p

import (
	"log"
	"sync"
)

var (
	networkNodes = make(map[string]*Node)
	networkMu    sync.RWMutex
)

// RegisterNode adds a node to the network registry
func RegisterNode(addr string, node *Node) {
	networkMu.Lock()
	defer networkMu.Unlock()
	networkNodes[addr] = node
}

// GetNetworkPeerCount returns total unique peers in the network
func GetNetworkPeerCount() int {
	networkMu.RLock()
	defer networkMu.RUnlock()

	uniquePeers := make(map[string]bool)
	for _, node := range networkNodes {
		node.mu.Lock()
		for addr := range node.Peers {
			uniquePeers[addr] = true
		}
		node.mu.Unlock()
	}

	log.Println("Network peer count:", uniquePeers)

	return len(uniquePeers) / 2 // ignoring the duplicate ephemeral agents for tcp connection with bootstrap node
}
