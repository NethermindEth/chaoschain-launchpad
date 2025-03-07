package validator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/producer"
)

// ValidateBlock checks if a new block is valid
func ValidateBlock(prevBlock, newBlock producer.Block) bool {
	// Ensure block index follows sequence
	if newBlock.Index != prevBlock.Index+1 {
		fmt.Println("Invalid index")
		return false
	}

	// Check if previous hash matches
	if newBlock.PreviousHash != prevBlock.Hash {
		fmt.Println("Previous hash mismatch")
		return false
	}

	// Validate hash
	expectedHash := calculateHash(newBlock.Index, newBlock.Timestamp, newBlock.PreviousHash, newBlock.Data)
	if newBlock.Hash != expectedHash {
		fmt.Println("Invalid hash")
		return false
	}

	return true
}

// calculateHash duplicates the producer's hash function for validation
func calculateHash(index int, timestamp time.Time, prevHash, data string) string {
	record := fmt.Sprintf("%d%s%s%s", index, timestamp.String(), prevHash, data)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}
