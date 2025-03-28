package p2p

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// PeerStoreType represents the type of peer storage/discovery mechanism
type PeerStoreType string

const (
	PeerStoreMemory  PeerStoreType = "memory"  // In-memory peer storage
	PeerStoreFile    PeerStoreType = "file"    // File-based peer storage
	PeerStoreService PeerStoreType = "service" // Discovery service
)

// PeerStore manages discovered peers
type PeerStore struct {
	knownPeers  map[string]time.Time // Address -> last seen
	storeType   PeerStoreType
	storePath   string
	mutex       sync.RWMutex
	maxPeerAge  time.Duration // Maximum time to keep a peer without refreshing
	seedNodes   []string      // Always try to connect to seed nodes
	environment string        // dev, test, or prod
}

// DefaultPeerStore is the global peer store
var DefaultPeerStore = NewPeerStore(PeerStoreMemory, "")

// NewPeerStore creates a new peer store
func NewPeerStore(storeType PeerStoreType, path string) *PeerStore {
	ps := &PeerStore{
		knownPeers:  make(map[string]time.Time),
		storeType:   storeType,
		storePath:   path,
		maxPeerAge:  24 * time.Hour,
		environment: "dev", // Default to dev environment
	}

	// Set appropriate seed nodes based on environment
	ps.setSeedNodesByEnvironment()

	// Initialize store based on type
	ps.initialize()

	return ps
}

// setSeedNodesByEnvironment sets appropriate seed nodes based on environment
func (ps *PeerStore) setSeedNodesByEnvironment() {
	switch ps.environment {
	case "dev":
		ps.seedNodes = []string{
			"localhost:8081",
			"localhost:8082",
		}
	case "test":
		ps.seedNodes = []string{
			"test-seed-1.chaoschain.example:9001",
			"test-seed-2.chaoschain.example:9001",
		}
	case "prod":
		ps.seedNodes = []string{
			"seed-1.chaoschain.io:9001",
			"seed-2.chaoschain.io:9001",
			"seed-3.chaoschain.io:9001",
			"seed-4.chaoschain.io:9001",
		}
	}
}

// SetEnvironment changes the environment and updates seed nodes
func (ps *PeerStore) SetEnvironment(env string) {
	ps.environment = env
	ps.setSeedNodesByEnvironment()
}

// initialize sets up the peer store based on its type
func (ps *PeerStore) initialize() {
	switch ps.storeType {
	case PeerStoreFile:
		ps.loadPeersFromFile()
	case PeerStoreService:
		ps.fetchPeersFromService()
	}
}

// loadPeersFromFile loads peers from a file
func (ps *PeerStore) loadPeersFromFile() {
	// Implementation depends on file format
	// For now, just log the intent
	log.Printf("Would load peers from %s", ps.storePath)
}

// fetchPeersFromService gets peers from a discovery service
func (ps *PeerStore) fetchPeersFromService() {
	// In production, this would contact a discovery service
	// For now, just add the seed nodes
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	now := time.Now()
	for _, seed := range ps.seedNodes {
		ps.knownPeers[seed] = now
	}
}

// AddPeer adds a peer to the store
func (ps *PeerStore) AddPeer(addr string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.knownPeers[addr] = time.Now()

	// In a production system, we'd periodically persist changes
	// For now, just log the addition
	log.Printf("Added peer %s to peer store", addr)
}

// GetPeers returns a list of known peers
func (ps *PeerStore) GetPeers(limit int) []string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	now := time.Now()
	var validPeers []string

	// First add all seed nodes
	for _, seed := range ps.seedNodes {
		validPeers = append(validPeers, seed)
	}

	// Then add other peers that aren't too old
	for addr, lastSeen := range ps.knownPeers {
		// Skip if already added (from seedNodes)
		if contains(validPeers, addr) {
			continue
		}

		// Skip if too old
		if now.Sub(lastSeen) > ps.maxPeerAge {
			continue
		}

		validPeers = append(validPeers, addr)

		// Stop if we've reached the limit
		if limit > 0 && len(validPeers) >= limit {
			break
		}
	}

	// Shuffle to avoid always connecting to the same subset
	rand.Shuffle(len(validPeers), func(i, j int) {
		validPeers[i], validPeers[j] = validPeers[j], validPeers[i]
	})

	return validPeers
}

// RemovePeer removes a peer from the store
func (ps *PeerStore) RemovePeer(addr string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	delete(ps.knownPeers, addr)
}

// UpdatePeer updates the last seen time for a peer
func (ps *PeerStore) UpdatePeer(addr string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.knownPeers[addr] = time.Now()
}

// CleanupOldPeers removes peers that haven't been seen recently
func (ps *PeerStore) CleanupOldPeers() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	now := time.Now()
	for addr, lastSeen := range ps.knownPeers {
		// Skip seed nodes
		if contains(ps.seedNodes, addr) {
			continue
		}

		if now.Sub(lastSeen) > ps.maxPeerAge {
			delete(ps.knownPeers, addr)
			log.Printf("Removed stale peer %s from peer store", addr)
		}
	}
}

// Helper to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// DiscoverPeers attempts to connect to known peers
func (n *Node) DiscoverPeers() {
	// Get peers from the peer store
	peers := DefaultPeerStore.GetPeers(MAX_PEERS)

	log.Printf("Discovering peers from %d candidates", len(peers))

	for _, peer := range peers {
		n.ConnectToPeer(peer)
	}

	// Request peer lists from connected peers (PEX)
	n.RequestPeerExchange()
}

// RequestPeerExchange sends a request to connected peers for their peer lists
func (n *Node) RequestPeerExchange() {
	pexRequest := NewMessage("GET_PEERS", nil)

	n.mu.RLock()
	peerCount := len(n.Peers)
	n.mu.RUnlock()

	if peerCount == 0 {
		return // No peers to ask
	}

	n.BroadcastMessage(pexRequest)
	log.Println("Requested peer exchange from connected peers")
}

// HandlePeerExchange processes a peer list from another peer
func (n *Node) HandlePeerExchange(peerList []string) {
	// Add new peers to the peer store
	for _, addr := range peerList {
		// Don't add self
		myAddr := fmt.Sprintf("localhost:%d", n.port)
		if addr == myAddr {
			continue
		}

		DefaultPeerStore.AddPeer(addr)
	}

	// Connect to new peers if needed
	n.mu.RLock()
	currentPeerCount := len(n.Peers)
	n.mu.RUnlock()

	if currentPeerCount < MIN_PEERS {
		n.DiscoverPeers()
	}
}
