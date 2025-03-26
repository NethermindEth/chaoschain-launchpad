package handlers

import (
	"fmt"
	"sync"
)

type NodeInfo struct {
	IsGenesis bool
	RPCPort   int
	P2PPort   int
}

var (
	// Map of chainID -> Map of nodeID -> NodeInfo
	chainNodes    = make(map[string]map[string]NodeInfo)
	registryMutex sync.RWMutex
)

func RegisterNode(chainID string, nodeID string, info NodeInfo) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	if _, exists := chainNodes[chainID]; !exists {
		chainNodes[chainID] = make(map[string]NodeInfo)
	}
	chainNodes[chainID][nodeID] = info
}

func getRPCPortForChain(chainID string) (int, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	nodes, exists := chainNodes[chainID]
	if !exists {
		return 0, fmt.Errorf("chain %s not found", chainID)
	}

	// Find genesis node
	for _, info := range nodes {
		if info.IsGenesis {
			return info.RPCPort, nil
		}
	}

	return 0, fmt.Errorf("genesis node not found for chain %s", chainID)
}
