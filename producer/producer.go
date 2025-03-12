package producer

import (
	"encoding/json"
	"log"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

// Producer handles block production in the system.
type Producer struct {
	Mempool     core.MempoolInterface
	Personality ai.Personality
	LastBlock   *core.Block // Keeps track of last block for chaining
	p2pNode     *p2p.Node
}

// NewProducer initializes a block Producer.
func NewProducer(mp core.MempoolInterface, personality ai.Personality, p2pNode *p2p.Node) *Producer {
	return &Producer{
		Mempool:     mp,
		Personality: personality,
		LastBlock:   nil,
		p2pNode:     p2pNode,
	}
}

// ProduceBlock creates a new block, signs it, and publishes its proposal both via NATS and TCP-based P2P.
func (p *Producer) ProduceBlock() core.Block {
	prevHash := "genesis"
	height := 1
	if p.LastBlock != nil {
		prevHash = p.LastBlock.Hash()
		height = p.LastBlock.Height + 1
	}

	// Select transactions from the mempool.
	txs := p.Mempool.GetPendingTransactions()
	selectedTxs := p.Personality.SelectTransactions(txs)

	// Create a new block.
	block := core.Block{
		Height:    height,
		PrevHash:  prevHash,
		Txs:       selectedTxs,
		Timestamp: time.Now().Unix(),
		Signature: "", // TODO: Implement AI-based cryptographic signing
	}

	// Let the AI generate a block signature.
	block.Signature = p.Personality.SignBlock(block)

	// Remove processed transactions.
	for _, tx := range selectedTxs {
		p.Mempool.RemoveTransaction(tx.Signature)
	}

	announcement := p.Personality.GenerateBlockAnnouncement(block)
	log.Printf("New Block (Height: %d) Announcement: %s", block.Height, announcement)

	// Marshal block data to JSON.
	blockBytes, err := json.Marshal(block)
	if err != nil {
		log.Printf("Error encoding block proposal: %v", err)
	} else {
		// Publish block proposal over NATS.
		err = core.NatsBrokerInstance.Publish("BLOCK_PROPOSAL", blockBytes)
		if err != nil {
			log.Printf("Error publishing BLOCK_PROPOSAL via NATS: %v", err)
		} else {
			log.Println("Published BLOCK_PROPOSAL event via NATS")
		}
	}

	// Also broadcast block proposal over the TCP-based P2P layer.
	// Note: p2p.GetP2PNode().BroadcastMessage sends a p2p.Message that includes the block.
	p2p.GetP2PNode().BroadcastMessage(p2p.Message{
		Type: "BLOCK_PROPOSAL",
		Data: block,
	})
	log.Println("Broadcasted BLOCK_PROPOSAL event via P2P layer")

	p.LastBlock = &block
	return block
}
