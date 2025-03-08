package mempool

import (
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/core"
)

// Mempool stores pending transactions before they are added to a block
type Mempool struct {
	mu            sync.Mutex
	transactions  map[string]core.Transaction
	expirationSec int64 // Transactions expire after X seconds
}

var defaultMempool *Mempool

// Initialize mempool separately
func InitMempool(expirationSec int64) {
	defaultMempool = &Mempool{
		transactions:  make(map[string]core.Transaction),
		expirationSec: expirationSec,
	}
}

// GetMempool returns the default mempool instance
func GetMempool() core.MempoolInterface {
	return defaultMempool
}

// AddTransaction adds a new transaction to the mempool if valid
func (mp *Mempool) AddTransaction(tx core.Transaction) bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Ensure transaction is valid before adding
	if !tx.VerifyTransaction(tx.From) {
		return false
	}

	mp.transactions[tx.Signature] = tx
	return true
}

// GetPendingTransactions returns all pending transactions
func (mp *Mempool) GetPendingTransactions() []core.Transaction {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	var txs []core.Transaction
	for _, tx := range mp.transactions {
		txs = append(txs, tx)
	}
	return txs
}

// RemoveTransaction removes a transaction once it's included in a block
func (mp *Mempool) RemoveTransaction(txID string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	delete(mp.transactions, txID)
}

// CleanupExpiredTransactions removes old transactions
func (mp *Mempool) CleanupExpiredTransactions() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	now := time.Now().Unix()
	for id, tx := range mp.transactions {
		if now-tx.Timestamp > mp.expirationSec {
			delete(mp.transactions, id)
		}
	}
}

// Size returns the number of transactions in the mempool
func (mp *Mempool) Size() int {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return len(mp.transactions)
}
