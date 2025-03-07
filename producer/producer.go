package producer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Block represents a simple blockchain block
type Block struct {
	Index        int
	Timestamp    time.Time
	PreviousHash string
	Data         string
	Hash         string
}

// NewBlock creates a new block
func NewBlock(index int, prevHash, data string) Block {
	timestamp := time.Now()
	hash := calculateHash(index, timestamp, prevHash, data)

	return Block{
		Index:        index,
		Timestamp:    timestamp,
		PreviousHash: prevHash,
		Data:         data,
		Hash:         hash,
	}
}

// calculateHash generates a SHA-256 hash for a block
func calculateHash(index int, timestamp time.Time, prevHash, data string) string {
	record := fmt.Sprintf("%d%s%s%s", index, timestamp.String(), prevHash, data)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}
