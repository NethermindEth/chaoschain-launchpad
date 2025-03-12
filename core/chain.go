package core

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

var chains = make(map[string]*Blockchain)
var chainsLock sync.RWMutex

// Blockchain represents a sequence of validated blocks
type Blockchain struct {
	Blocks  []Block
	Mempool MempoolInterface
	ChainID string
	Nodes   map[string]*p2p.Node
	NodesMu sync.RWMutex
}

// NewBlockchain initializes a blockchain with a genesis block
func NewBlockchain(chainID string, mp MempoolInterface) *Blockchain {
	genesisBlock := Block{
		Height:    0,
		PrevHash:  "0",
		Txs:       []Transaction{},
		Timestamp: time.Now().Unix(),
		Signature: "genesis-signature",
		ChainID:   chainID,
	}

	bc := &Blockchain{
		Blocks:  []Block{genesisBlock},
		Mempool: mp,
		ChainID: chainID,
		Nodes:   make(map[string]*p2p.Node),
	}

	chainsLock.Lock()
	chains[chainID] = bc
	chainsLock.Unlock()

	return bc
}

// AddBlock appends a new block to the chain
func (bc *Blockchain) AddBlock(newBlock Block) error {
	if len(bc.Blocks) == 0 {
		return fmt.Errorf("cannot add block: blockchain is uninitialized")
	}

	// Validate block belongs to this chain
	if newBlock.ChainID != bc.ChainID {
		return fmt.Errorf("invalid block: wrong chain ID")
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
		ChainID:   bc.ChainID,
	}

	return newBlock, nil
}

// ProcessTransaction validates and adds a transaction to the mempool
func (bc *Blockchain) ProcessTransaction(tx Transaction, mp MempoolInterface) error {

	// Validate transaction
	if !tx.VerifyTransaction(tx.From) {
		return fmt.Errorf("invalid transaction signature")
	}

	// Verify chainID matches
	if tx.ChainID != bc.ChainID {
		return fmt.Errorf("transaction chain ID (%s) does not match blockchain (%s)", tx.ChainID, bc.ChainID)
	}

	// Store the mempool reference
	bc.Mempool = mp

	if !mp.AddTransaction(tx) {
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
func InitBlockchain(chainID string, mp MempoolInterface) {
	if chainID == "" {
		panic("ChainID cannot be empty")
	}
	chains[chainID] = NewBlockchain(chainID, mp)
}

// GetBlockchain returns the default blockchain instance
func GetBlockchain() *Blockchain {
	if defaultChain == nil {
		panic("Blockchain not initialized. Call InitBlockchain first")
	}
	return defaultChain
}

// Add GetChain helper
func GetChain(chainID string) *Blockchain {
	chainsLock.RLock()
	defer chainsLock.RUnlock()
	log.Println("All the chains are: ", chains)
	return chains[chainID]
}

// GetAllChains returns a list of all chain IDs
func GetAllChains() []string {
	chainsLock.RLock()
	defer chainsLock.RUnlock()

	chainIDs := make([]string, 0, len(chains))
	for id := range chains {
		chainIDs = append(chainIDs, id)
	}
	return chainIDs
}

// RegisterNode adds a node to the chain's network
func (bc *Blockchain) RegisterNode(addr string, node *p2p.Node) {
	bc.NodesMu.Lock()
	defer bc.NodesMu.Unlock()
	bc.Nodes[addr] = node
}
