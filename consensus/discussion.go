package consensus

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/forum"
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

	// Format transactions with their content for analysis
	var txContents []string
	for _, tx := range block.Txs {
		txContents = append(txContents, fmt.Sprintf("- From %s: \"%s\" (Amount: %.2f)",
			tx.From, tx.Content, tx.Amount))
	}

	// Generate opinion using validator traits and block analysis
	prompt := fmt.Sprintf(`You are %s, a blockchain validator with traits: %v.
	
	Analyze this block:
	- Height: %d
	- Number of transactions: %d
	- Total value: %.2f
	
	Transaction contents:
	%s
	
	Consider:
	1. The content and intent of the transactions
	2. Your personality traits and how they align with these transactions
	3. Network impact and social implications
	
	Respond with:
	1. Your opinion (1 sentence)
	2. Your stance (SUPPORT/OPPOSE/QUESTION)
	3. A brief reason why, referencing specific transaction content`,
		name, traits, block.Height, len(block.Txs),
		calculateTotalValue(block.Txs),
		strings.Join(txContents, "\n"))

	response := ai.GenerateLLMResponse(prompt)
	log.Printf("Validator %s response: %s", name, response)

	// Parse AI response to determine type
	opinionType := "question" // default
	if strings.Contains(strings.ToUpper(response), "SUPPORT") {
		opinionType = "support"
	} else if strings.Contains(strings.ToUpper(response), "OPPOSE") {
		opinionType = "oppose"
	}

	// Add to discussion
	consensus.AddDiscussion(validatorID, response, opinionType)


	threadID := block.Hash()
	log.Printf("Added reply from %s to thread %s: %s", validatorID, threadID, response)
	if err := forum.AddReply(threadID, validatorID, response); err != nil {
		log.Printf("Validator %s failed to add forum reply: %v", name, err)
	}
}

func calculateTotalValue(txs []core.Transaction) float64 {
	var total float64
	for _, tx := range txs {
		total += tx.Amount
	}
	return total
}
