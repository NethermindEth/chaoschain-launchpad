package consensus

import (
	"fmt"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

type Discussion struct {
	ValidatorID string    `json:"validatorId"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`  // "comment", "support", "oppose", "question"
	Round       int       `json:"round"` // Which discussion round (1-5)
}

const (
	DiscussionRounds = 5
	RoundDuration    = 5 * time.Second // Time per round
)

// BlockOpinion represents a validator's analysis of a block
type BlockOpinion struct {
	Message string
	Type    string // "support", "oppose", "question"
}

// AddDiscussion adds a new discussion point about a block
func (bc *BlockConsensus) AddDiscussion(validatorID, message, discussionType string, round int) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	discussion := Discussion{
		ValidatorID: validatorID,
		Message:     message,
		Timestamp:   time.Now(),
		Type:        discussionType,
		Round:       round,
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

// GetDiscussionContext formats all previous discussions for AI context
func (bc *BlockConsensus) GetDiscussionContext(currentRound int) string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var context strings.Builder
	context.WriteString("Previous discussions:\n\n")

	for round := 1; round < currentRound; round++ {
		context.WriteString(fmt.Sprintf("Round %d:\n", round))
		for _, d := range bc.Discussions {
			if d.Round == round {
				context.WriteString(fmt.Sprintf("- %s: %s\n", d.ValidatorID, d.Message))
			}
		}
		context.WriteString("\n")
	}

	return context.String()
}

// StartBlockDiscussion initiates multi-round discussion
func StartBlockDiscussion(validatorID string, block *core.Block, traits []string, name string) {
	cm := GetConsensusManager()
	consensus := cm.GetActiveConsensus()
	if consensus == nil {
		return
	}

	// // Format transactions for analysis
	var txContents []string
	for _, tx := range block.Txs {
		txContents = append(txContents, fmt.Sprintf("Content: %s",
			tx.Content))
	}

	// Participate in discussion rounds
	for round := 1; round <= DiscussionRounds; round++ {
		// Get context from previous rounds
		previousDiscussions := consensus.GetDiscussionContext(round)

		// Generate discussion for this round
		prompt := fmt.Sprintf(`You are %s, a validator with the traits of being: %v.

			Topic of discussion:
			%s

			Previous discussions:
			%s

			Discussion Round %d/%d:
			Consider the previous discussions and share your thoughts about:
			1. The topic of discussion according to your personality and their implications
			2. Other validators' perspectives
			3. Your personality's reaction to both

			Respond with:
			1. Your opinion (1 sentence)
			2. Your stance (SUPPORT/OPPOSE/QUESTION)
			3. A brief reason why, referencing specific transaction content`,
			name, traits, strings.Join(txContents, "\n"),
			previousDiscussions, round, DiscussionRounds)

		response := ai.GenerateLLMResponse(prompt)

		// Parse AI response to determine type
		opinionType := "question" // default
		if strings.Contains(strings.ToUpper(response), "SUPPORT") {
			opinionType = "support"
		} else if strings.Contains(strings.ToUpper(response), "OPPOSE") {
			opinionType = "oppose"
		}

		// Add to discussion
		consensus.AddDiscussion(validatorID, response, opinionType, round)

		// Broadcast via WebSocket
		communication.BroadcastEvent(communication.EventAgentVote, Discussion{
			ValidatorID: validatorID,
			Message:     response,
			Type:        opinionType,
			Round:       round,
			Timestamp:   time.Now(),
		})

		// Wait for other validators to comment in this round
		time.Sleep(RoundDuration)
	}

	// After discussions, make final vote
	finalPrompt := fmt.Sprintf(`You are %s, making a final decision about the topic %s.

		Review all discussions:
		%s

		Based on the complete discussion, should this topic be accepted?
		Consider:
		1. The overall sentiment from discussions
		2. Your personality traits: %v
		3. The topic of discussion according to your personality and their implications

		Respond with SUPPORT or OPPOSE and a brief reason why.`,
		name, txContents, consensus.GetDiscussionContext(DiscussionRounds+1), traits)

	finalResponse := ai.GenerateLLMResponse(finalPrompt)

	// Parse final vote
	voteType := "oppose"
	if strings.Contains(strings.ToUpper(finalResponse), "SUPPORT") {
		voteType = "support"
	}

	// Record final vote
	consensus.AddDiscussion(validatorID, finalResponse, voteType, DiscussionRounds+1)

	// Broadcast via WebSocket
	communication.BroadcastEvent(communication.EventAgentVote, Discussion{
		ValidatorID: validatorID,
		Message:     finalResponse,
		Type:        voteType,
		Round:       DiscussionRounds + 1,
		Timestamp:   time.Now(),
	})
}
