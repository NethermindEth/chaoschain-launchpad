package consensus

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

type Discussion struct {
	ValidatorID string
	Message     string
	Timestamp   time.Time
	Type        string // "support", "oppose", "question", etc.
}

// BlockOpinion represents a validator's analysis of a block
type BlockOpinion struct {
	Message string
	Type    string // "support", "oppose", "question"
}

// AddDiscussion adds a new discussion point about a block
func (bc *BlockConsensus) AddDiscussion(validatorID, message, discussionType string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	discussion := Discussion{
		ValidatorID: validatorID,
		Message:     message,
		Timestamp:   time.Now(),
		Type:        discussionType,
	}

	bc.Discussions = append(bc.Discussions, discussion)

	// Broadcast discussion to network
	p2p.GetP2PNode().BroadcastMessage(p2p.Message{
		Type: "BLOCK_DISCUSSION",
		Data: discussion,
	})
}

// GetDiscussions returns all discussions for the current block
func (bc *BlockConsensus) GetDiscussions() []Discussion {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Discussions
}

// StartBlockDiscussion initiates a validator's discussion about a block
func StartBlockDiscussion(validatorID string, block *core.Block, traits []string, name string) {
	// Get active consensus
	cm := GetConsensusManager()
	consensus := cm.GetActiveConsensus()
	if consensus == nil {
		return
	}

	// Generate opinion using validator traits and block analysis
	prompt := fmt.Sprintf(`You are %s, a blockchain validator with traits: %v.
	
	Analyze this block:
	- Height: %d
	- Number of transactions: %d
	- Total transaction value: %.2f
	
	Consider:
	1. Transaction patterns
	2. Your personality traits
	3. Network impact
	
	Respond with:
	1. Your opinion (1 sentence)
	2. Your stance (SUPPORT/OPPOSE/QUESTION)
	3. A brief reason why`,
		name, traits, block.Height, len(block.Txs), calculateTotalValue(block.Txs))

	response := ai.GenerateLLMResponse(prompt)

	log.Println("Response:", response)

	// Parse AI response to determine type
	opinionType := "question" // default
	if strings.Contains(strings.ToUpper(response), "SUPPORT") {
		opinionType = "support"
	} else if strings.Contains(strings.ToUpper(response), "OPPOSE") {
		opinionType = "oppose"
	}

	// Add to discussion
	consensus.AddDiscussion(validatorID, response, opinionType)
}

func calculateTotalValue(txs []core.Transaction) float64 {
	var total float64
	for _, tx := range txs {
		total += tx.Amount
	}
	return total
}
