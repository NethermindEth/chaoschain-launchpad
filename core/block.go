package core

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/crypto"
)

// Block represents a basic block structure
type Block struct {
	Height    int           `json:"height"`
	PrevHash  string        `json:"prev_hash"`
	Txs       []Transaction `json:"transactions"`
	Timestamp int64         `json:"timestamp"`
	Signature string        `json:"signature"`
	Proposer  string        `json:"proposer"`
	ChainID   string        `json:"chain_id"`
}

var (
	latestBlock Block
	mutex       sync.Mutex
)

func GetLatestBlockHeight() int {
	mutex.Lock()
	defer mutex.Unlock()
	return latestBlock.Height
}

func GetLatestBlockHash() string {
	mutex.Lock()
	defer mutex.Unlock()
	return latestBlock.PrevHash
}

func SetLatestBlock(block Block) {
	mutex.Lock()
	defer mutex.Unlock()
	latestBlock = block
}

// SignBlock signs a block using the validator's private key
func (b *Block) SignBlock(privateKey string) error {
	b.Timestamp = time.Now().Unix() // Set timestamp
	b.Signature = ""                // Reset signature before signing

	blockData, err := json.Marshal(b)
	if err != nil {
		return err
	}

	signature, err := crypto.SignMessage(privateKey, blockData)
	if err != nil {
		return err
	}

	b.Signature = signature
	return nil
}

// VerifyBlock verifies the authenticity of a block
func (b *Block) VerifyBlock(publicKey string) bool {
	signature := b.Signature
	b.Signature = "" // Remove signature before verifying

	blockData, _ := json.Marshal(b)
	b.Signature = signature // Restore signature after verification

	return crypto.VerifySignature(publicKey, string(blockData), signature)
}

// Hash returns the block's hash
func (b *Block) Hash() string {
	blockData, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return crypto.HashData(string(blockData))
}
