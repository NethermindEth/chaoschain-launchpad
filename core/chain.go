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
	// Only validate height and previous hash
	if block.Height <= 0 || block.PrevHash == "" {
		return false
	}

	// For now, allow empty blocks and don't check signatures
	// TODO: Add proper block signing and validation
	return true
}

// GetBlockByHeight retrieves a block at a specific height
func GetBlockByHeight(height int) (Block, bool) {
	if height < 0 || height >= len(defaultChain.Blocks) {
		return Block{}, false
	}
	return defaultChain.Blocks[height], true
}

// CreateBlock creates a new block proposal (doesn't add to chain)
func (bc *Blockchain) CreateBlock() (*Block, error) {
	if len(bc.Blocks) == 0 {
		return nil, fmt.Errorf("blockchain not initialized")
	}

	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// Get pending transactions from mempool
	pendingTxs := bc.Mempool.GetPendingTransactions()
	if len(pendingTxs) == 0 {
		return nil, fmt.Errorf("no pending transactions")
	}

	// Create new block
	newBlock := &Block{
		Height:    lastBlock.Height + 1,
		PrevHash:  lastBlock.Hash(),
		Txs:       pendingTxs,
		Timestamp: time.Now().Unix(),
		Signature: "temp", // TODO: Add proper block signing
	}

	return newBlock, nil
}

// ProcessTransaction validates and adds a transaction to the mempool
func (bc *Blockchain) ProcessTransaction(tx Transaction) error {
	// Validate transaction
	if !tx.VerifyTransaction(tx.From) {
		return fmt.Errorf("invalid transaction signature")
	}

	// Add to mempool
	if !bc.Mempool.AddTransaction(tx) {
		return fmt.Errorf("failed to add transaction to mempool")
	}

	// Broadcast transaction
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
