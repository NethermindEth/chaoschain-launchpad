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

// Blockchain represents a sequence of validated blocks along with its genesis prompt.
type Blockchain struct {
	Blocks        []Block
	GenesisPrompt string // New field: stores the chain's genesis prompt.
	Mempool       MempoolInterface
	ChainID       string
	Nodes         map[string]*p2p.Node
	NodesMu       sync.RWMutex
	RewardPool    int
}

// NewBlockchain initializes a blockchain with a genesis block and the given genesis prompt.
func NewBlockchain(chainID string, mp MempoolInterface, genesisPrompt string, rewardPool int) *Blockchain {
	genesisBlock := Block{
		Height:    0,
		PrevHash:  "0",
		Txs:       []Transaction{},
		Timestamp: time.Now().Unix(),
		Signature: "genesis-signature",
		ChainID:   chainID,
	}
	bc := &Blockchain{
		Blocks:        []Block{genesisBlock},
		GenesisPrompt: genesisPrompt,
		Mempool:       mp,
		ChainID:       chainID,
		Nodes:         make(map[string]*p2p.Node),
		RewardPool:    rewardPool,
	}
	chainsLock.Lock()
	chains[chainID] = bc
	chainsLock.Unlock()
	return bc
}

// AddBlock appends a new block to the chain.
func (bc *Blockchain) AddBlock(newBlock Block) error {
	if len(bc.Blocks) == 0 {
		return fmt.Errorf("cannot add block: blockchain is uninitialized")
	}
	// Validate block belongs to this chain.
	if newBlock.ChainID != bc.ChainID {
		return fmt.Errorf("invalid block: wrong chain ID")
	}
	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	// Ensure the block links properly.
	if newBlock.PrevHash != lastBlock.Hash() {
		return fmt.Errorf("invalid block: previous hash mismatch")
	}
	// Validate block.
	if !bc.ValidateBlock(newBlock) {
		return fmt.Errorf("invalid block: validation failed")
	}
	bc.Blocks = append(bc.Blocks, newBlock)
	return nil
}

// ValidateBlock checks whether a given block follows chain rules.
func (bc *Blockchain) ValidateBlock(block Block) bool {
	if block.Height <= 0 || block.PrevHash == "" {
		return false
	}
	return true
}

// GetBlockByHeight retrieves a block at a specific height.
func GetBlockByHeight(height int) (Block, bool) {
	if height < 0 || height >= len(defaultChain.Blocks) {
		return Block{}, false
	}
	return defaultChain.Blocks[height], true
}

// CreateBlock creates a new block proposal (without adding it to the chain).
func (bc *Blockchain) CreateBlock() (*Block, error) {
	if len(bc.Blocks) == 0 {
		return nil, fmt.Errorf("blockchain not initialized")
	}
	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	pendingTxs := bc.Mempool.GetPendingTransactions()
	if len(pendingTxs) == 0 {
		return nil, fmt.Errorf("no pending transactions")
	}
	newBlock := &Block{
		Height:    lastBlock.Height + 1,
		PrevHash:  lastBlock.Hash(),
		Txs:       pendingTxs,
		Timestamp: time.Now().Unix(),
		Signature: "temp", // TODO: Add proper block signing.
		ChainID:   bc.ChainID,
	}
	return newBlock, nil
}

// ProcessTransaction validates and adds a transaction to the mempool.
func (bc *Blockchain) ProcessTransaction(tx Transaction, mp MempoolInterface) error {
	if !tx.VerifyTransaction(tx.From) {
		return fmt.Errorf("invalid transaction signature")
	}
	if tx.ChainID != bc.ChainID {
		return fmt.Errorf("transaction chain ID (%s) does not match blockchain (%s)", tx.ChainID, bc.ChainID)
	}
	bc.Mempool = mp
	if !mp.AddTransaction(tx) {
		return fmt.Errorf("failed to add transaction to mempool")
	}
	txData, _ := json.Marshal(tx)
	p2p.GetP2PNode().BroadcastMessage(p2p.Message{
		Type: "TRANSACTION",
		Data: string(txData),
	})
	return nil
}

var defaultChain *Blockchain

// InitBlockchain initializes the blockchain with the provided genesis prompt.
func InitBlockchain(chainID string, mp MempoolInterface, genesisPrompt string, rewardPool int) {
	if chainID == "" {
		panic("ChainID cannot be empty")
	}
	chains[chainID] = NewBlockchain(chainID, mp, genesisPrompt, rewardPool)
}

// GetBlockchain returns the default blockchain instance.
func GetBlockchain() *Blockchain {
	if defaultChain == nil {
		panic("Blockchain not initialized. Call InitBlockchain first")
	}
	return defaultChain
}

// GetChain returns the blockchain associated with the given chainID.
func GetChain(chainID string) *Blockchain {
	chainsLock.RLock()
	defer chainsLock.RUnlock()
	log.Println("All the chains are: ", chains)
	return chains[chainID]
}

type ChainInfo struct {
	ChainID string `json:"chain_id"`
	Name    string `json:"name"`
	Agents  int    `json:"agents"`
	Blocks  int    `json:"blocks"`
}

// GetAllChains returns a list of all chain IDs.
func GetAllChains() []ChainInfo {
	chainsLock.RLock()
	defer chainsLock.RUnlock()
	chainInfos := make([]ChainInfo, 0, len(chains))
	for id, chain := range chains {
		chainInfos = append(chainInfos, ChainInfo{
			ChainID: id,
			Name:    id,                   // Using chainID as name for now.
			Agents:  len(chain.Nodes) - 1, // Subtract 1 to exclude bootstrap node.
			Blocks:  len(chain.Blocks),
		})
	}
	return chainInfos
}

// RegisterNode adds a node to the chain's network.
func (bc *Blockchain) RegisterNode(addr string, node *p2p.Node) {
	bc.NodesMu.Lock()
	defer bc.NodesMu.Unlock()
	bc.Nodes[addr] = node
}
