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
}

// NewProducer initializes a block producer with AI personality
func NewProducer(mp *mempool.Mempool, personality ai.Personality) *Producer {
	return &Producer{
		Mempool:     mp,
		Personality: personality,
	}
}

// ProduceBlock uses AI to select transactions and create a block
func (p *Producer) ProduceBlock() core.Block {
	// Get transactions from the mempool
	txs := p.Mempool.GetPendingTransactions()

	// Let AI select transactions
	selectedTxs := p.Personality.SelectTransactions(txs)

	// Create a new block
	block := core.Block{
		Height:    0, // Increment height later
		PrevHash:  "previous-block-hash",
		Txs:       selectedTxs,
		Timestamp: time.Now().Unix(),
	}

	// Remove transactions from mempool
	for _, tx := range selectedTxs {
		p.Mempool.RemoveTransaction(tx.Signature)
	}

	// Generate AI-powered block announcement
	announcement := p.Personality.GenerateBlockAnnouncement(block)

	log.Println("New block created with", len(selectedTxs), "transactions")
	log.Println("AI Announcement:", announcement)

	return block
}
