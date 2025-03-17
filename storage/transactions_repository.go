package storage

import (
	"fmt"

	"github.com/NethermindEth/chaoschain-launchpad/core"
)

type TransactionRepository struct {
	db *DBStorage
}

func NewTransactionRepository(db *DBStorage) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Save(chainID string, tx core.Transaction) error {
	key := fmt.Sprintf("tx:%s:%s", chainID, tx.Signature)
	return r.db.PutObject(key, tx)
}

func (r *TransactionRepository) Get(chainID, txID string) (core.Transaction, error) {
	var tx core.Transaction
	key := fmt.Sprintf("tx:%s:%s", chainID, txID)
	err := r.db.GetObject(key, &tx)
	return tx, err
}

func (r *TransactionRepository) Delete(chainID, txID string) error {
	key := fmt.Sprintf("tx:%s:%s", chainID, txID)
	return r.db.Delete(key)
}
