package core

// MempoolInterface defines the required functionality for a mempool
type MempoolInterface interface {
	AddTransaction(tx interface{}) bool
	GetPendingTransactions() []Transaction
	RemoveTransaction(txID string)
	CleanupExpiredTransactions()
	Size() int
}
