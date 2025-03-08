package core

// ValidationResult represents the outcome of block validation
type ValidationResult struct {
	BlockHash string `json:"block_hash"`
	Valid     bool   `json:"valid"`
	Reason    string `json:"reason"`
	Meme      string `json:"meme"`
}
