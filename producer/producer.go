package producer

import (
	"log"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
)

// Producer handles block production in ChaosChain
type Producer struct {
	Mempool     *mempool.Mempool
	Personality ai.Personality
	LastBlock   *core.Block // Keeps track of last block for chaining
}

// NewProducer initializes a block producer with AI personality
func NewProducer(mp *mempool.Mempool, personality ai.Personality) *Producer {
	return &Producer{
		Mempool:     mp,
		Personality: personality,
		LastBlock:   nil, // No previous block at the start
	}
}

// ProduceBlock uses AI to select transactions and create a block
func (p *Producer) ProduceBlock() core.Block {
	// Get previous block hash (default to "genesis" if no previous block)
	prevHash := "genesis"
	height := 1
	if p.LastBlock != nil {
		prevHash = p.LastBlock.Hash()
		height = p.LastBlock.Height + 1
	}

	// Get transactions from the mempool
	txs := p.Mempool.GetPendingTransactions()

	// Let AI select transactions
	selectedTxs := p.Personality.SelectTransactions(txs)

	// Create a new block
	block := core.Block{
		Height:    height,
		PrevHash:  prevHash,
		Txs:       selectedTxs,
		Timestamp: time.Now().Unix(),
		Signature: "", // TODO: Implement AI-based cryptographic signing
	}

	// AI Generates a signature for the block (to be verified by validators)
	block.Signature = p.Personality.SignBlock(block)

	// Remove transactions from mempool
	for _, tx := range selectedTxs {
		p.Mempool.RemoveTransaction(tx.Signature)
	}

	// Generate AI-powered block announcement
	announcement := p.Personality.GenerateBlockAnnouncement(block)

	log.Printf("🔷 New Block Created (Height: %d, Txns: %d)", block.Height, len(selectedTxs))
	log.Printf("📢 AI Announcement: %s", announcement)

	// Store the last created block
	p.LastBlock = &block

	return block
}
