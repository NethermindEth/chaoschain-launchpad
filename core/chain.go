package core

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

// Blockchain represents a sequence of validated blocks
type Blockchain struct {
	Blocks  []Block
	Mempool MempoolInterface
}

// NewBlockchain initializes a blockchain with a genesis block
func NewBlockchain(mp MempoolInterface) *Blockchain {
	genesisBlock := Block{
		Height:    0,
		PrevHash:  "0",
		Txs:       []Transaction{},
		Timestamp: time.Now().Unix(),
		Signature: "genesis-signature",
	}

	return &Blockchain{
		Blocks:  []Block{genesisBlock},
		Mempool: mp, // Use interface instead of direct dependency
	}
}

// AddBlock appends a new block to the chain
func (bc *Blockchain) AddBlock(newBlock Block) error {
	if len(bc.Blocks) == 0 {
		return fmt.Errorf("cannot add block: blockchain is uninitialized")
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// Ensure the block links properly
	if newBlock.PrevHash != lastBlock.Hash() {
		return fmt.Errorf("invalid block: previous hash mismatch")
	}

	// Validate the block before adding
	if !bc.ValidateBlock(newBlock) {
		return fmt.Errorf("invalid block: validation failed")
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
}

// ValidateBlock checks whether a given block follows chain rules
func (bc *Blockchain) ValidateBlock(block Block) bool {
	// Ensure block has valid structure
	if block.Height <= 0 || block.PrevHash == "" || len(block.Txs) == 0 {
		return false
	}

	// Check block hash integrity
	return block.Hash() == block.Signature
}

// GetBlockByHeight retrieves a block at a specific height
func GetBlockByHeight(height int) (Block, bool) {
	if height < 0 || height >= len(defaultChain.Blocks) {
		return Block{}, false
	}
	return defaultChain.Blocks[height], true
}

// GetNetworkStatus returns the current blockchain status
func GetNetworkStatus() map[string]interface{} {
	return map[string]interface{}{
		"height":     len(defaultChain.Blocks) - 1,
		"latestHash": defaultChain.Blocks[len(defaultChain.Blocks)-1].Hash(),
		"totalTxs":   len(defaultChain.Blocks[len(defaultChain.Blocks)-1].Txs),
		// Let the API layer handle peer count
	}
}

// ProcessTransaction validates and adds a transaction to the mempool
func (bc *Blockchain) ProcessTransaction(tx Transaction) error {
	// Validate transaction
	if !tx.VerifyTransaction(tx.From) {
		return fmt.Errorf("invalid transaction signature")
	}

	// Add to mempool using the interface
	if !bc.Mempool.AddTransaction(tx) {
		return fmt.Errorf("failed to add transaction to mempool")
	}

	// Broadcast to network
	txData, _ := json.Marshal(tx)
	p2p.GetP2PNode().BroadcastMessage(p2p.Message{
		Type: "TRANSACTION",
		Data: string(txData),
	})

	return nil
}

var defaultChain *Blockchain

// Initialize blockchain
func InitBlockchain(mp MempoolInterface) {
	defaultChain = NewBlockchain(mp)
}

// GetBlockchain returns the default blockchain instance
func GetBlockchain() *Blockchain {
	if defaultChain == nil {
		panic("Blockchain not initialized. Call InitBlockchain first")
	}
	return defaultChain
}
