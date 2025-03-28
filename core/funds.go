package core

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ChainFunds represents the available funds in a chain
type ChainFunds struct {
	ChainID    string             `json:"chain_id"`
	TotalFunds float64            `json:"total_funds"`
	Balances   map[string]float64 `json:"balances"` // validatorID -> balance
	mutex      sync.RWMutex
}

var (
	// Map of chainID -> ChainFunds
	chainFundsRegistry = make(map[string]*ChainFunds)
	registryMutex      sync.RWMutex
)

// InitializeChainFunds creates a new chain with initial funds
func InitializeChainFunds(chainID string, initialFunds float64) *ChainFunds {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	// Check if chain funds already exist
	if funds, exists := chainFundsRegistry[chainID]; exists {
		log.Printf("Chain funds for %s already initialized with %.2f funds", chainID, funds.TotalFunds)
		return funds
	}

	funds := &ChainFunds{
		ChainID:    chainID,
		TotalFunds: initialFunds,
		Balances:   make(map[string]float64),
	}

	chainFundsRegistry[chainID] = funds
	log.Printf("Initialized chain %s with %.2f initial funds", chainID, initialFunds)
	return funds
}

// GetChainFunds returns the funds for a specific chain
func GetChainFunds(chainID string) *ChainFunds {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	return chainFundsRegistry[chainID]
}

// GetBalance returns the balance for a specific validator
func (cf *ChainFunds) GetBalance(validatorID string) float64 {
	cf.mutex.RLock()
	defer cf.mutex.RUnlock()

	return cf.Balances[validatorID]
}

// AddFunds adds funds to the chain's total
func (cf *ChainFunds) AddFunds(amount float64) {
	cf.mutex.Lock()
	defer cf.mutex.Unlock()

	cf.TotalFunds += amount
	log.Printf("Added %.2f funds to chain %s, new total: %.2f", amount, cf.ChainID, cf.TotalFunds)
}

// ProcessRewards processes a reward transaction and distributes funds to validators
func (cf *ChainFunds) ProcessRewards(tx *Transaction, recipients map[string]float64) error {
	cf.mutex.Lock()
	defer cf.mutex.Unlock()

	// Calculate total reward amount
	totalReward := tx.Reward

	// Ensure chain has enough funds
	if totalReward > cf.TotalFunds {
		return fmt.Errorf("insufficient chain funds (%.2f) for reward (%.2f)", cf.TotalFunds, totalReward)
	}

	// Distribute rewards
	for validatorID, amount := range recipients {
		// Update validator balance
		cf.Balances[validatorID] += amount
		log.Printf("Rewarded validator %s with %.2f funds", validatorID, amount)
	}

	// Deduct from total funds
	cf.TotalFunds -= totalReward
	log.Printf("Processed reward transaction of %.2f funds, remaining chain funds: %.2f",
		totalReward, cf.TotalFunds)

	return nil
}

// CreateRewardTransaction creates a special transaction to reward validators
func CreateRewardTransaction(
	proposerID string,
	chainID string,
	totalReward float64,
	recipients map[string]float64) *Transaction {

	// Create a transaction with reward details
	tx := &Transaction{
		From:      "CHAIN",    // Special sender indicating it's a chain reward
		To:        proposerID, // Primary recipient is the proposer
		Amount:    0,          // No direct amount transfer
		Fee:       0,          // No fee for reward transactions
		Type:      "REWARD",
		Content:   fmt.Sprintf("Block reward distribution of %.2f funds", totalReward),
		Timestamp: GetCurrentTimestamp(),
		ChainID:   chainID,
		Reward:    totalReward,
	}

	return tx
}

// ValidateRewardTransaction validates that a reward transaction is valid
func ValidateRewardTransaction(tx *Transaction, chainID string) bool {
	// Ensure it's a reward transaction
	if tx.Type != "REWARD" {
		return false
	}

	// Ensure it's for the correct chain
	if tx.ChainID != chainID {
		return false
	}

	// Ensure the reward is positive
	if tx.Reward <= 0 {
		return false
	}

	// Ensure it's from the chain
	if tx.From != "CHAIN" {
		return false
	}

	return true
}

// GetCurrentTimestamp returns the current Unix timestamp
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}
