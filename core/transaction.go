package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Transaction represents a basic transaction structure
type Transaction struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Fee       uint64  `json:"fee"` // 💰 New: Transaction fee
	Content   string  `json:"content"`
	Timestamp int64   `json:"timestamp"`
	Signature string  `json:"signature"`
	PublicKey string  `json:"publicKey"`
	ChainID   string  `json:"chainID"`
	Type      string  `json:"type"`
	Reward    float64 `json:"reward"`
}

// GenerateKeyPair creates a new key pair for signing transactions
func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// SignTransaction signs a transaction with the given private key
func (tx *Transaction) SignTransaction(privateKey *ecdsa.PrivateKey) error {
	// Create hash of transaction data
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%s%.8f", tx.From, tx.To, tx.Amount)))

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return err
	}

	// Store signature and public key
	tx.Signature = hex.EncodeToString(append(r.Bytes(), s.Bytes()...))
	tx.PublicKey = hex.EncodeToString(elliptic.MarshalCompressed(privateKey.PublicKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y))

	return nil
}

// VerifyTransaction verifies the transaction signature
func (tx *Transaction) VerifyTransaction(from string) bool {
	// TODO: In the final implementation, we would:
	// 1. Decode the signature and public key
	// 2. Recreate the transaction hash
	// 3. Verify the signature using the public key

	// For now, just verify the sender matches
	return tx.From == from
}
