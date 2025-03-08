package core

import (
	"encoding/json"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/crypto"
)

// Transaction represents a basic transaction structure
type Transaction struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Fee       uint64  `json:"fee"` // 💰 New: Transaction fee
	Timestamp int64   `json:"timestamp"`
	Signature string  `json:"signature"`
}

// SignTransaction signs a transaction using the sender's private key
func (tx *Transaction) SignTransaction(privateKey string) error {
	tx.Timestamp = time.Now().Unix()
	tx.Signature = ""

	txData, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	signature, err := crypto.SignMessage(privateKey, txData)
	if err != nil {
		return err
	}

	tx.Signature = signature
	return nil
}

// VerifyTransaction verifies the authenticity of a transaction
func (tx *Transaction) VerifyTransaction(publicKey string) bool {
	signature := tx.Signature
	tx.Signature = ""

	txData, _ := json.Marshal(tx)
	tx.Signature = signature

	return crypto.VerifySignature(publicKey, string(txData), signature)
}
