package p2p

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
)

// Peer represents a node in the P2P network
type Peer struct {
	Address string
	Conn    net.Conn
}

// Node manages peer connections and message handling
type Node struct {
	Peers       map[string]*Peer
	mu          sync.Mutex
	listener    net.Listener
	subscribers map[string][]func([]byte)
}

var defaultNode = NewNode()

// GetP2PNode returns the default P2P node instance
func GetP2PNode() *Node {
	return defaultNode
}

// NewNode initializes a new P2P network node
func NewNode() *Node {
	return &Node{
		Peers:       make(map[string]*Peer),
		subscribers: make(map[string][]func([]byte)),
	}
}

// StartServer starts listening for new connections
func (p *Node) StartServer(port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	p.listener = listener
	log.Printf("P2P server started on port %d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Connection failed: %v", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

// ConnectToPeer connects to a peer at a given address
func (p *Node) ConnectToPeer(address string) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", address, err)
		return
	}

	peer := &Peer{Address: address, Conn: conn}
	p.mu.Lock()
	p.Peers[address] = peer
	p.mu.Unlock()

	go p.listenToPeer(peer)
	log.Printf("Connected to peer: %s\n", address)
}

// handleConnection handles incoming peer connections
func (p *Node) handleConnection(conn net.Conn) {
	peer := &Peer{Address: conn.RemoteAddr().String(), Conn: conn}
	p.mu.Lock()
	p.Peers[peer.Address] = peer
	p.mu.Unlock()

	go p.listenToPeer(peer)
}

// listenToPeer listens for messages from a peer
func (p *Node) listenToPeer(peer *Peer) {
	defer peer.Conn.Close()

	for {
		buffer := make([]byte, 4096)
		n, err := peer.Conn.Read(buffer)
		if err != nil {
			log.Printf("Connection lost with %s", peer.Address)
			p.mu.Lock()
			delete(p.Peers, peer.Address)
			p.mu.Unlock()
			return
		}

		var msg Message
		err = json.Unmarshal(buffer[:n], &msg)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		go p.handleMessage(msg, peer)
	}
}

// handleMessage processes incoming messages
func (p *Node) handleMessage(msg Message, peer *Peer) {
	log.Printf("Received message from %s: %s", peer.Address, msg.Type)

	switch msg.Type {
	case "TRANSACTION":
		// Process incoming transaction
		log.Println("Transaction received:", msg.Data)
	case "BLOCK":
		// Process new block
		log.Println("New block received:", msg.Data)
	case "VALIDATION":
		// Process validation result
		log.Println("Validation received:", msg.Data)
	}
}

// BroadcastMessage sends a message to all peers
func (p *Node) BroadcastMessage(msg Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	msgBytes, _ := json.Marshal(msg)
	for _, peer := range p.Peers {
		_, err := peer.Conn.Write(msgBytes)
		if err != nil {
			log.Printf("Failed to send message to %s: %v", peer.Address, err)
		}
	}
}

// Subscribe registers a callback for a specific message type
func (p *Node) Subscribe(msgType string, callback func([]byte)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.subscribers[msgType] = append(p.subscribers[msgType], callback)
}

// Publish sends a message to all subscribers of a specific type
func (p *Node) Publish(msgType string, data []byte) {
	p.mu.Lock()
	callbacks := p.subscribers[msgType]
	p.mu.Unlock()

	for _, callback := range callbacks {
		go callback(data)
	}
}
