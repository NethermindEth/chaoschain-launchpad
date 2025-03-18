package handlers

import (
	"fmt"
	"sync"
)

type chainRegistry struct {
	mu     sync.RWMutex
	chains map[string]int // chainID -> RPC port
}

var registry = &chainRegistry{
	chains: make(map[string]int),
}

func registerChainPort(chainID string, port int) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.chains[chainID] = port
}

func getRPCPortForChain(chainID string) (int, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	port, exists := registry.chains[chainID]
	if !exists {
		return 0, fmt.Errorf("chain %s not registered", chainID)
	}
	return port, nil
}

func extractPortFromAddress(address string) int {
	var port int
	fmt.Sscanf(address, "tcp://0.0.0.0:%d", &port)
	return port
}
