package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Peer represents a node in the P2P network
type Peer struct {
	Address string
	Conn    net.Conn
	AgentID AgentID // New field to track the agent identity
	// Add fields for connection quality metrics
	LastSeen time.Time
	Latency  time.Duration // Track average message latency
}

// ChainConfig represents the configuration for a specific chain
type ChainConfig struct {
	ChainID    string
	P2PPort    int
	APIPort    int
	NetworkKey string // Optional: Could be used to further isolate networks
}

// Node manages peer connections and message handling
type Node struct {
	ChainID       string
	AgentID       AgentID // Unique identity for this node's agent
	Peers         map[string]*Peer
	mu            sync.RWMutex // Changed from sync.Mutex to sync.RWMutex
	listener      net.Listener
	subscribers   map[string][]func([]byte)
	port          int
	seenMessages  map[MessageID]bool          // Track already processed messages
	msgMu         sync.RWMutex                // Separate mutex for message tracking
	directMsgSubs map[AgentID][]func(Message) // Subscribers for direct messages
	security      *SecurityProvider           // Added security provider for crypto operations
}

var defaultNode = NewNode(ChainConfig{ChainID: "main", P2PPort: 8080})

// GetP2PNode returns the default P2P node instance
func GetP2PNode() *Node {
	return defaultNode
}

// NewNode initializes a new P2P network node
func NewNode(config ChainConfig) *Node {
	// Generate a unique agent ID for this node
	agentID := AgentID(GenerateUUID())

	// Initialize security provider
	security := NewSecurityProvider()

	node := &Node{
		ChainID:       config.ChainID,
		AgentID:       agentID,
		Peers:         make(map[string]*Peer),
		subscribers:   make(map[string][]func([]byte)),
		port:          config.P2PPort,
		seenMessages:  make(map[MessageID]bool),
		directMsgSubs: make(map[AgentID][]func(Message)),
		security:      security,
	}

	// Try to initialize security with a key file
	keyDir := "./keys"
	security.LoadOrCreateKeyPair(keyDir, string(agentID))

	return node
}

// StartServer starts listening for new connections
func (n *Node) StartServer(port int) {
	n.port = port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	n.listener = listener
	log.Printf("P2P server started on port %d\n", port)

	// Server is ready to accept connections
	go n.acceptConnections()

	// Start maintenance routine
	go n.maintainConnections()
}

// maintainConnections periodically checks peer connections and cleans up dead ones
func (n *Node) maintainConnections() {
	// Run peer maintenance every 30 seconds
	maintenanceTicker := time.NewTicker(30 * time.Second)

	// Run peer rotation every 5 minutes to ensure network diversity
	rotationTicker := time.NewTicker(5 * time.Minute)

	// Run full cleanup every hour
	cleanupTicker := time.NewTicker(1 * time.Hour)

	defer func() {
		maintenanceTicker.Stop()
		rotationTicker.Stop()
		cleanupTicker.Stop()
	}()

	for {
		select {
		case <-maintenanceTicker.C:
			// Basic maintenance - ping peers and clean up dead ones
			n.cleanupDeadPeers()

			// Try to discover new peers if below minimum
			if n.GetPeerCount() < MIN_PEERS {
				n.DiscoverPeers()
			}

		case <-rotationTicker.C:
			// Rotate some peers to maintain network diversity
			n.rotatePeers()

		case <-cleanupTicker.C:
			// Clean up message tracking (remove older messages to prevent memory leaks)
			n.cleanupSeenMessages()

			// Clean up old peers from the peer store
			DefaultPeerStore.CleanupOldPeers()
		}
	}
}

// rotatePeers disconnects from some peers and connects to new ones
// This helps maintain network diversity and avoids network clustering
func (n *Node) rotatePeers() {
	n.mu.RLock()
	peerCount := len(n.Peers)
	// Don't rotate if we don't have enough peers
	if peerCount <= MIN_PEERS {
		n.mu.RUnlock()
		return
	}

	// Get a list of non-seed peers to consider for rotation
	var candidatesForRotation []string
	for addr := range n.Peers {
		// Skip seed nodes - keep them connected
		isSeed := false
		for _, seed := range DefaultPeerStore.seedNodes {
			if seed == addr {
				isSeed = true
				break
			}
		}

		if !isSeed {
			candidatesForRotation = append(candidatesForRotation, addr)
		}
	}
	n.mu.RUnlock()

	// Calculate how many peers to rotate (20% of peers, but at least 1)
	rotationCount := len(candidatesForRotation) / 5
	if rotationCount < 1 && len(candidatesForRotation) > 0 {
		rotationCount = 1
	}

	// Shuffle candidates and take the first rotationCount
	rand.Shuffle(len(candidatesForRotation), func(i, j int) {
		candidatesForRotation[i], candidatesForRotation[j] = candidatesForRotation[j], candidatesForRotation[i]
	})

	// Select peers to disconnect
	for i := 0; i < rotationCount && i < len(candidatesForRotation); i++ {
		addr := candidatesForRotation[i]
		log.Printf("Rotating peer: disconnecting from %s", addr)

		n.mu.Lock()
		if peer, exists := n.Peers[addr]; exists {
			peer.Conn.Close() // Will trigger cleanup in listenToPeer
		}
		n.mu.Unlock()
	}

	// After disconnection, discover new peers
	time.Sleep(1 * time.Second) // Brief delay to allow disconnect handling
	n.DiscoverPeers()
}

// cleanupSeenMessages removes old messages from tracking
func (n *Node) cleanupSeenMessages() {
	const messageExpiration = 5 * time.Minute

	n.msgMu.Lock()
	defer n.msgMu.Unlock()

	// In a real implementation, we would use timestamps
	// For simplicity, just cap the map size here
	if len(n.seenMessages) > 10000 {
		n.seenMessages = make(map[MessageID]bool)
	}
}

// cleanupDeadPeers removes disconnected peers
func (n *Node) cleanupDeadPeers() {
	deadPeers := []string{}

	n.mu.Lock()
	for addr, peer := range n.Peers {
		// Send a ping message
		pingMsg := NewMessage("PING", nil)
		pingMsg.SenderID = n.AgentID // Assign directly instead of using SetSender
		msgBytes, _ := json.Marshal(pingMsg)

		if _, err := peer.Conn.Write(msgBytes); err != nil {
			deadPeers = append(deadPeers, addr)
		}
	}

	// Remove dead peers
	for _, addr := range deadPeers {
		delete(n.Peers, addr)
		log.Printf("Removed dead peer: %s", addr)
	}
	n.mu.Unlock()
}

func (n *Node) acceptConnections() {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			log.Printf("Connection failed: %v", err)
			continue
		}
		go n.handleConnection(conn)
	}
}

// Add these constants
const (
	MAX_PEERS = 10 // Maximum number of peer connections
	MIN_PEERS = 3  // Minimum desired peer connections
)

// Add handshake struct at package level
type handshakeMsg struct {
	ChainID   string `json:"chain_id"`
	Address   string `json:"address"`
	AgentID   string `json:"agent_id"`   // Agent identity
	PublicKey string `json:"public_key"` // Base64 encoded public key
	Version   string `json:"version"`    // Protocol version for compatibility
	NodeType  string `json:"node_type"`  // The type of node (validator, producer, etc.)
	Timestamp int64  `json:"timestamp"`  // Handshake timestamp
}

// ConnectToPeer connects to a peer at a given address
func (n *Node) ConnectToPeer(address string) {
	myAddr := fmt.Sprintf("localhost:%d", n.port)

	// Don't connect to self
	if address == myAddr {
		return
	}

	// Don't connect if we already have this peer
	n.mu.RLock()
	if _, exists := n.Peers[address]; exists {
		n.mu.RUnlock()
		return
	}
	n.mu.RUnlock()

	log.Printf("Node %s attempting to connect to peer at %s", myAddr, address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", address, err)
		return
	}

	// Export public key if available
	var publicKeyStr string
	if n.security != nil && n.security.keyPair != nil {
		publicKeyStr, _ = n.security.ExportPublicKey()
	}

	// Send handshake
	handshake := handshakeMsg{
		ChainID:   n.ChainID,
		Address:   myAddr,
		AgentID:   string(n.AgentID),
		PublicKey: publicKeyStr,
		Version:   "1.0.0",
		NodeType:  "generic", // Can be specialized based on node type
		Timestamp: time.Now().Unix(),
	}

	handshakeData, _ := json.Marshal(handshake)
	if _, err := conn.Write(handshakeData); err != nil {
		conn.Close()
		return
	}

	// Wait for handshake response
	buffer := make([]byte, 4096) // Larger buffer for handshake with public key
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		return
	}

	var response handshakeMsg
	if err := json.Unmarshal(buffer[:bytesRead], &response); err != nil {
		conn.Close()
		return
	}

	// Verify chain ID
	if response.ChainID != n.ChainID {
		conn.Close()
		return
	}

	// Create peer with the remote agent ID
	peer := &Peer{
		Address:  address,
		Conn:     conn,
		AgentID:  AgentID(response.AgentID),
		LastSeen: time.Now(),
	}

	// Register peer's public key if provided
	if response.PublicKey != "" && n.security != nil {
		publicKey, err := n.security.ImportPublicKey(response.PublicKey)
		if err != nil {
			log.Printf("Warning: Failed to import public key from peer %s: %v", response.AgentID, err)
		} else {
			n.security.RegisterPublicKey(response.AgentID, publicKey)
			log.Printf("Registered public key for agent %s", response.AgentID)
		}
	}

	n.mu.Lock()
	n.Peers[address] = peer
	n.mu.Unlock()

	go n.listenToPeer(peer)
	log.Printf("Node %s connected to peer: %s (Agent: %s)\n", myAddr, address, peer.AgentID)
}

// handleConnection handles incoming peer connections
func (n *Node) handleConnection(conn net.Conn) {
	// Read initial handshake
	buffer := make([]byte, 4096) // Larger buffer for handshake with public key
	bytesRead, err := conn.Read(buffer)
	if err != nil {
		conn.Close()
		return
	}

	var handshake handshakeMsg
	if err := json.Unmarshal(buffer[:bytesRead], &handshake); err != nil {
		log.Printf("Invalid handshake from incoming connection: %v", err)
		conn.Close()
		return
	}

	// Verify chain ID
	if handshake.ChainID != n.ChainID {
		log.Printf("Rejecting peer from different chain: %s", handshake.ChainID)
		conn.Close()
		return
	}

	// Verify protocol version compatibility
	if handshake.Version != "" && !isVersionCompatible(handshake.Version, "1.0.0") {
		log.Printf("Rejecting peer with incompatible protocol version: %s", handshake.Version)
		conn.Close()
		return
	}

	myAddr := fmt.Sprintf("localhost:%d", n.port)

	// Use the address sent in handshake
	peerAddr := handshake.Address
	peerAgentID := AgentID(handshake.AgentID)

	// Only accept connection if we don't have this peer and it's not ourselves
	n.mu.Lock()
	if _, exists := n.Peers[peerAddr]; exists || peerAddr == myAddr {
		n.mu.Unlock()
		conn.Close()
		return
	}

	peer := &Peer{
		Address:  peerAddr,
		Conn:     conn,
		AgentID:  peerAgentID,
		LastSeen: time.Now(),
	}
	n.Peers[peerAddr] = peer
	n.mu.Unlock()

	// Add peer to peer store
	DefaultPeerStore.AddPeer(peerAddr)

	// Register peer's public key if provided
	if handshake.PublicKey != "" && n.security != nil {
		publicKey, err := n.security.ImportPublicKey(handshake.PublicKey)
		if err != nil {
			log.Printf("Warning: Failed to import public key from peer %s: %v", handshake.AgentID, err)
		} else {
			n.security.RegisterPublicKey(handshake.AgentID, publicKey)
			log.Printf("Registered public key for agent %s", handshake.AgentID)
		}
	}

	// Prepare our public key if available
	var publicKeyStr string
	if n.security != nil && n.security.keyPair != nil {
		publicKeyStr, _ = n.security.ExportPublicKey()
	}

	// Send handshake response
	response := handshakeMsg{
		ChainID:   n.ChainID,
		Address:   myAddr,
		AgentID:   string(n.AgentID),
		PublicKey: publicKeyStr,
		Version:   "1.0.0",
		NodeType:  "generic", // Can be specialized based on node type
		Timestamp: time.Now().Unix(),
	}
	handshakeData, _ := json.Marshal(response)
	conn.Write(handshakeData)

	go n.listenToPeer(peer)
	log.Printf("Node %s accepted connection from: %s (Agent: %s)\n", myAddr, peerAddr, peer.AgentID)
}

// isVersionCompatible checks if two semantic versions are compatible
func isVersionCompatible(version1, version2 string) bool {
	// For now, simple string comparison
	// In production, we should parse versions and check major/minor compatibility
	return version1 == version2
}

// listenToPeer listens for messages from a peer
func (n *Node) listenToPeer(peer *Peer) {
	defer peer.Conn.Close()

	for {
		buffer := make([]byte, 4096)
		bytesRead, err := peer.Conn.Read(buffer)
		if err != nil {
			log.Printf("Connection lost with %s", peer.Address)
			n.mu.Lock()
			delete(n.Peers, peer.Address)
			n.mu.Unlock()
			return
		}

		// Update last seen timestamp
		peer.LastSeen = time.Now()

		var msg Message
		err = json.Unmarshal(buffer[:bytesRead], &msg)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Check if we've seen this message before (prevents loops)
		n.msgMu.RLock()
		seen := n.seenMessages[msg.ID]
		n.msgMu.RUnlock()

		if seen {
			// Skip processing if we've seen this message
			continue
		}

		// Mark as seen
		n.msgMu.Lock()
		n.seenMessages[msg.ID] = true
		n.msgMu.Unlock()

		go n.handleMessage(msg, peer)
	}
}

// handleMessage processes incoming messages
func (n *Node) handleMessage(msg Message, peer *Peer) {
	log.Printf("Received message from %s (Agent: %s): %s", peer.Address, peer.AgentID, msg.Type)

	// Verify message signature if available
	if msg.Signature != nil && len(msg.Signature) > 0 && n.security != nil {
		// Only verify if we know the sender's public key
		if _, exists := n.security.knownPublicKeys[string(msg.SenderID)]; exists {
			verified, err := n.security.VerifyMessageSignature(msg)
			if err != nil {
				log.Printf("Warning: Failed to verify message signature: %v", err)
			} else if !verified {
				log.Printf("Warning: Message signature verification failed, possible forgery from %s", msg.SenderID)
				// In production, we might want to take more severe action like temporarily banning the peer
			}
		}
	}

	// Handle directed messages
	if msg.IsDirected() && msg.RecipientID == n.AgentID {
		// This message is specifically for this node's agent
		log.Printf("Received direct message from %s to me", msg.SenderID)

		// Notify subscribers for this direct message
		n.mu.RLock()
		callbacks := n.directMsgSubs[msg.SenderID]
		n.mu.RUnlock()

		for _, callback := range callbacks {
			go callback(msg)
		}

		// Also deliver to type-specific subscribers if any
		data, ok := msg.Data.([]byte)
		if ok {
			n.Publish(msg.Type, data)
		}
		return
	} else if msg.IsDirected() && msg.RecipientID != n.AgentID {
		// This message is for another agent - relay it if TTL allows
		if msg.TTL > 0 {
			// Decrement TTL and forward
			msg.TTL--
			n.RelayMessage(msg)
		}
		return
	}

	// Update peer's last seen time in the peer store
	DefaultPeerStore.UpdatePeer(peer.Address)

	// Handle broadcast messages
	switch msg.Type {
	case "PING":
		// Respond to ping with pong
		pongMsg := NewMessage("PONG", nil)
		pongMsg.SenderID = n.AgentID
		n.SendToPeer(peer, pongMsg)

	case "PONG":
		// Calculate and store latency
		if msg.Timestamp.IsZero() {
			// Can't calculate latency without timestamp
			return
		}
		latency := time.Since(msg.Timestamp)
		peer.Latency = latency

	case "GET_PEERS":
		// Send our peer list
		n.mu.RLock()
		peerList := make([]string, 0, len(n.Peers))
		for addr := range n.Peers {
			peerList = append(peerList, addr)
		}
		n.mu.RUnlock()

		response := NewMessage("PEER_LIST", peerList)
		response.SenderID = n.AgentID
		n.SendToPeer(peer, response) // Direct response instead of broadcast

	case "PEER_LIST":
		// Process received peer list
		if peerList, ok := msg.Data.([]string); ok {
			n.HandlePeerExchange(peerList)
		}

	case "PUBLIC_KEY":
		// Handle public key exchange
		if keyData, ok := msg.Data.(string); ok {
			n.handlePublicKeyExchange(string(msg.SenderID), keyData)
		}
	}

	// Publish to type-specific subscribers
	data, ok := msg.Data.([]byte)
	if ok {
		n.Publish(msg.Type, data)
	}
}

// handlePublicKeyExchange processes a received public key
func (n *Node) handlePublicKeyExchange(senderID, encodedKey string) {
	if n.security == nil {
		log.Println("Security provider not available, skipping public key import")
		return
	}

	// Import the public key
	publicKey, err := n.security.ImportPublicKey(encodedKey)
	if err != nil {
		log.Printf("Failed to import public key from %s: %v", senderID, err)
		return
	}

	// Register the public key
	n.security.RegisterPublicKey(senderID, publicKey)
	log.Printf("Imported and registered public key from agent %s", senderID)
}

// ExchangePublicKey sends this node's public key to a peer
func (n *Node) ExchangePublicKey(peer *Peer) error {
	if n.security == nil || n.security.keyPair == nil {
		return errors.New("security provider not available or no key pair")
	}

	// Export public key
	encodedKey, err := n.security.ExportPublicKey()
	if err != nil {
		return err
	}

	// Create and send message
	msg := NewMessage("PUBLIC_KEY", encodedKey)
	msg.SenderID = n.AgentID

	return n.SendToPeer(peer, msg)
}

// SendToPeer sends a message to a specific peer
func (n *Node) SendToPeer(peer *Peer, msg Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = peer.Conn.Write(msgBytes)
	return err
}

// SendDirectMessage sends a message directly to a specific agent
func (n *Node) SendDirectMessage(recipientID AgentID, msgType string, data interface{}) error {
	msg := NewMessage(msgType, data)
	msg.SenderID = n.AgentID
	msg.RecipientID = recipientID

	// Find a peer that has this agent ID
	n.mu.RLock()
	var targetPeer *Peer
	for _, peer := range n.Peers {
		if peer.AgentID == recipientID {
			targetPeer = peer
			break
		}
	}
	n.mu.RUnlock()

	if targetPeer != nil {
		// Direct connection exists
		return n.SendToPeer(targetPeer, msg)
	}

	// No direct connection, broadcast with recipient specified
	n.BroadcastMessage(msg)
	return nil
}

// RelayMessage forwards a message to other peers
func (n *Node) RelayMessage(msg Message) {
	// Only relay if TTL > 0
	if msg.TTL <= 0 {
		return
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	// Send to all peers except the original sender
	for _, peer := range n.Peers {
		if peer.AgentID != msg.SenderID {
			peer.Conn.Write(msgBytes)
		}
	}
}

// BroadcastMessage sends a message to all peers
func (n *Node) BroadcastMessage(msg Message) {
	// Ensure the sender ID is set
	if msg.SenderID == "" {
		msg.SenderID = n.AgentID
	}

	// Ensure the message has an ID
	if msg.ID == "" {
		msg.ID = MessageID(GenerateUUID())
	}

	// Ensure timestamp is set
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Sign the message if security is available
	if n.security != nil && n.security.keyPair != nil {
		// Try to sign, but don't block sending if it fails
		if err := n.security.SignMessage(&msg); err != nil {
			log.Printf("Warning: Failed to sign message: %v", err)
		}
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, peer := range n.Peers {
		_, err := peer.Conn.Write(msgBytes)
		if err != nil {
			log.Printf("Failed to send message to %s: %v", peer.Address, err)
		}
	}
}

// Subscribe registers a callback for a specific message type
func (n *Node) Subscribe(msgType string, callback func([]byte)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subscribers[msgType] = append(n.subscribers[msgType], callback)
}

// SubscribeDirectMessages registers a callback for direct messages from a specific agent
func (n *Node) SubscribeDirectMessages(senderID AgentID, callback func(Message)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.directMsgSubs[senderID] = append(n.directMsgSubs[senderID], callback)
}

// Publish sends a message to all subscribers of a specific type
func (n *Node) Publish(msgType string, data []byte) {
	n.mu.RLock()
	callbacks := n.subscribers[msgType]
	n.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(data)
	}
}

func (n *Node) GetPort() int {
	return n.port
}

func SetDefaultNode(node *Node) {
	defaultNode = node
}

func (n *Node) GetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Peers)
}

// GetAgentID returns the AgentID of this node
func (n *Node) GetAgentID() AgentID {
	return n.AgentID
}
