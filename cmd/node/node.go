package node

import (
	"fmt"
	"log"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/registry"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

type NodeConfig struct {
	P2PPort       int
	APIPort       int
	BootstrapNode string
}

type Node struct {
	config   NodeConfig
	p2pNode  *p2p.Node
	mempool  core.MempoolInterface
	shutdown chan struct{}
}

func NewNode(config NodeConfig) *Node {
	return &Node{
		config:   config,
		p2pNode:  p2p.NewNode(config.P2PPort),
		shutdown: make(chan struct{}),
	}
}

func (n *Node) Start() error {
	// Initialize components
	mempool.InitMempool(3600)
	n.mempool = mempool.GetMempool()
	core.InitBlockchain(n.mempool)

	// Start P2P server
	log.Printf("Starting P2P node on port %d...", n.config.P2PPort)
	n.p2pNode.StartServer(n.config.P2PPort)

	// Register this node in the network
	addr := fmt.Sprintf("localhost:%d", n.config.P2PPort)
	p2p.RegisterNode(addr, n.p2pNode)

	// Set this as the default P2P node
	p2p.SetDefaultNode(n.p2pNode)

	// Give the server a moment to initialize
	time.Sleep(time.Second)

	// Connect to bootstrap node if provided
	if n.config.BootstrapNode != "" {
		n.p2pNode.ConnectToPeer(n.config.BootstrapNode)
	}

	return nil
}

func (n *Node) Stop() {
	close(n.shutdown)
}

func (n *Node) GetP2PPort() int {
	return n.config.P2PPort
}

func (n *Node) GetAPIPort() int {
	return n.config.APIPort
}

func (n *Node) GetP2PNode() *p2p.Node {
	return n.p2pNode
}

func (n *Node) GetMempool() core.MempoolInterface {
	return n.mempool
}

func (n *Node) RegisterProducer(id string, p *producer.Producer) {
	registry.RegisterProducer(id, p)
}

func (n *Node) RegisterValidator(id string, v *validator.Validator) {
	registry.RegisterValidator(id, v)
}
