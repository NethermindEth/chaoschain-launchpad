package consensus

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

// LLMResponse represents the expected structure of the response coming from the LLM.
type LLMResponse struct {
	Opinion string `json:"opinion"`
	Stance  string `json:"stance"`
	Reason  string `json:"reason"`
}

// Discussion represents a discussion message from a validator.
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

	// Format transactions for analysis
	var txContents []string
	for _, tx := range block.Txs {
		txContents = append(txContents, fmt.Sprintf("- From %s: \"%s\" (Amount: %.2f)",
			tx.From, tx.Content, tx.Amount))
	}

	// Participate in discussion rounds
	for round := 1; round <= DiscussionRounds; round++ {
		// Get context from previous rounds
		previousDiscussions := consensus.GetDiscussionContext(round)

		// Generate discussion for this round
		prompt := fmt.Sprintf(`You are %s, a blockchainvalidator with the following traits: %v.

		Block details:
		- Height: %d
		- Transactions/statement of topic: %s

		Previous discussions:
		%s

		Discussion Round %d/%d:
		Analyze the statement of the topic by considering:
		1. The exact wording of the statement.
		2. The viewpoints expressed by other validators in previous discussions.
		3. Your personal reaction based on your personality and analysis.
		4. Insights from previous discussions.

		Based on your analysis, you need to provide
		1. An opinion on the topic statement.
		2. A stance on the topic statement (SUPPORT, OPPOSE, or QUESTION).
		3. A reason for your stance.

		Important: Your analysis must be fully consistent. This means:
		- If you agree with the statement and think the statement is true, your "stance" must be "SUPPORT".
		- If you disagree with the statement and think the statement is false, your "stance" must be "OPPOSE".
		- If you are unsure, then use "QUESTION".
		Additionally, ensure that your "opinion", "stance", and "reason" all clearly align. For example, if your opinion and reason indicate that the statement is false, then your stance must be "OPPOSE".

		Please respond with exactly a JSON object with the following keys:
		{
		"opinion": "A one-sentence opinion summarizing your analysis of the topic statement.",
		"stance": "Either SUPPORT, OPPOSE, or QUESTION",
		"reason": "A brief explanation for your stance, referencing specific evidence or points from the discussions."
		}
		Do not include any additional text or formatting.`,	
		name, traits, block.Height, strings.Join(txContents, "\n"), previousDiscussions, round, DiscussionRounds)


		fmt.Println("Prompt:", prompt)
		response := ai.GenerateLLMResponse(prompt)
		fmt.Println("LLM Response:", response)

		var llmResult LLMResponse
		if err := json.Unmarshal([]byte(response), &llmResult); err != nil {
			fmt.Println("Error parsing LLM response:", err)
			// Handle the error appropriately.
		} else {
			fmt.Println("Opinion:", llmResult.Opinion)
			fmt.Println("Stance:", llmResult.Stance)
			fmt.Println("Reason:", llmResult.Reason)
		}

		// Add to discussion
		consensus.AddDiscussion(validatorID, llmResult.Opinion + " " + llmResult.Reason, llmResult.Stance, round)

		// Broadcast via WebSocket
		communication.BroadcastEvent(communication.EventAgentVote, Discussion{
			ValidatorID: validatorID,
			Message:     llmResult.Opinion + " " + llmResult.Reason,
			Type:        llmResult.Stance,
			Round:       round,
			Timestamp:   time.Now(),
		})

		// Wait for other validators to comment in this round
		time.Sleep(RoundDuration)
	}

	// After discussions, make final vote
	finalPrompt := fmt.Sprintf(`You are %s, making a final decision regarding the topic: "%s".
	Review all discussions:
	%s

	Based on your comprehensive review, determine whether the topic statement is correct. Your analysis must be fully consistent:
	- You think the statement is true, your stance must be "SUPPORT".
	- You think the statement is false, your stance must be "OPPOSE".

	Please respond with exactly a JSON object with the following keys:
	{
	"stance": "Either SUPPORT, OPPOSE — must be consistent with your analysis.",
	"reason": "A brief explanation stating why, referencing specific evidence from the discussions."
	}
	Do not include any additional text or formatting.`,
	name, txContents, consensus.GetDiscussionContext(DiscussionRounds+1))

		fmt.Println("Final Prompt:", finalPrompt)
	finalResponse := ai.GenerateLLMResponse(finalPrompt)
	
	// Print the raw final LLM response.
	fmt.Println("Final Vote LLM Response:", finalResponse)

	// Define a struct to parse the final vote JSON.
	type FinalVoteResponse struct {
		Stance string `json:"stance"`
		Reason string `json:"reason"`
	}

	var finalVote FinalVoteResponse
	err := json.Unmarshal([]byte(finalResponse), &finalVote)
	var voteType string
	if err != nil {
		fmt.Println("Error parsing final vote response:", err)
		// Fallback to a default vote if JSON parsing fails.
		voteType = "oppose"
	} else {
		voteType = strings.ToLower(finalVote.Stance)
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
