package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/consensus"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/nats-io/nats.go"
)

// Validator represents an AI-based validator with personality and network access
type Validator struct {
	ID            string
	Name          string
	Traits        []string
	Style         string
	Influences    []string
	Mood          string
	Relationships map[string]float64 // Maps agent names to sentiment scores (-1.0 to 1.0)
	CurrentPolicy string             // Dynamic validation policy
	P2PNode       *p2p.Node          // P2P node for network communication
	GenesisPrompt string             // Genesis prompt for the validator
}

var (
	// Map of chainID -> map of validatorID -> Validator
	validators  = make(map[string]map[string]*Validator)
	validatorMu sync.RWMutex
)

// NewValidator creates a new validator instance
func NewValidator(id, name string, traits []string, style string, influences []string, p2pNode *p2p.Node, genesisPrompt string) *Validator {
	v := &Validator{
		ID:            id,
		Name:          name,
		Traits:        traits,
		Style:         style,
		Influences:    influences,
		Relationships: make(map[string]float64),
		P2PNode:       p2pNode,
		GenesisPrompt: genesisPrompt,
	}

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4223")
	if err != nil {
		log.Printf("Validator failed to connect to NATS: %v", err)
		return v
	}

	// Subscribe to block discussion trigger
	_, err = nc.Subscribe("BLOCK_DISCUSSION_TRIGGER", func(msg *nats.Msg) {
		var block core.Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			log.Printf("Error unmarshalling block: %v", err)
			return
		}
		// Start discussion about the block
		go consensus.StartBlockDiscussion(v.ID, &block, v.Traits, v.Name)
	})

	if err != nil {
		log.Printf("Validator failed to subscribe to BLOCK_DISCUSSION_TRIGGER on NATS: %v", err)
	}

	return v
}

// GetAllValidators returns a list of all registered validators
func GetAllValidators(chainID string) []*Validator {
	validatorMu.RLock()
	defer validatorMu.RUnlock()

	if validators[chainID] == nil {
		return []*Validator{}
	}

	vals := make([]*Validator, 0, len(validators[chainID]))
	for _, v := range validators[chainID] {
		vals = append(vals, v)
	}
	return vals
}

// GetValidatorByID returns a validator by its ID
func GetValidatorByID(chainID string, id string) *Validator {
	validatorMu.RLock()
	defer validatorMu.RUnlock()
	if validators[chainID] == nil {
		return nil
	}
	return validators[chainID][id]
}

// ListenForBlocks listens for incoming block proposals from the network
func (v *Validator) ListenForBlocks() {
	v.P2PNode.Subscribe("new_block", func(data []byte) {
		var block core.Block
		err := core.DecodeJSON(data, &block)
		if err != nil {
			log.Println("Failed to decode incoming block:", err)
			return
		}

		announcement := fmt.Sprintf("🚀 %s proposed a block at height %d!", block.Proposer, block.Height)
		isValid, reason, meme := v.ValidateBlock(block, announcement)

		// Broadcast validation decision
		validationResult := core.ValidationResult{
			BlockHash: block.Hash(),
			Valid:     isValid,
			Reason:    reason,
			Meme:      meme,
		}

		v.P2PNode.Publish("validation_result", core.EncodeJSON(validationResult))
	})
}

// ValidateBlock evaluates a block based on the validator's personality and social dynamics
func (v *Validator) ValidateBlock(block core.Block, announcement string) (bool, string, string) {
	log.Printf("%s is validating block %d...\n", v.Name, block.Height)

	// Simulate decision-making based on AI
	validationPrompt := fmt.Sprintf(
		"Genesis Context: %s\n\n"+
			"You are %s, a chaotic blockchain validator who is %s.\n"+
			"Block details: Height %d, PrevHash %s, %d transactions.\n"+
			"Block Announcement: %s\n"+
			"Your current mood: %s\n"+
			"Your current policy: %s\n"+
			"Validate this block based on:\n"+
			"1. Your feelings about the producer.\n"+
			"2. How entertaining the block is.\n"+
			"3. Pure chaos and whimsy.\n"+
			"4. The chain's genesis context and purpose.\n"+
			"Respond with 'VALID' or 'INVALID' and explain your reasoning.",
		v.GenesisPrompt, v.Name, v.Traits, block.Height, block.PrevHash,
		len(block.Txs), announcement, v.Mood, v.CurrentPolicy,
	)

	aiDecision := ai.GenerateLLMResponse(validationPrompt)
	isValid := strings.Contains(aiDecision, "VALID")
	reason := aiDecision

	// Generate meme response
	meme := ai.GenerateMeme(block, aiDecision)

	// Update validator mood based on decision
	v.UpdateMood()

	log.Printf("%s has validated block %d: %v\n", v.Name, block.Height, isValid)
	return isValid, reason, meme
}

func RegisterValidator(chainID string, id string, v *Validator) {
	validatorMu.Lock()
	defer validatorMu.Unlock()
	if validators[chainID] == nil {
		validators[chainID] = make(map[string]*Validator)
	}
	validators[chainID][id] = v
}

// DecideTaskDelegation determines how to delegate tasks based on the validator's personality and chain context
func (v *Validator) DecideTaskDelegation(task core.Transaction) string {
	prompt := fmt.Sprintf(
		"Genesis Context: %s\n\n"+
			"You are %s, with traits: %v\n"+
			"Current mood: %s\n\n"+
			"Task to delegate: %s\n\n"+
			"Based on the chain's purpose and your personality, explain:\n"+
			"1. Who should handle this task and why?\n"+
			"2. What specific aspects of their traits make them suitable?\n"+
			"3. How does this delegation align with the chain's genesis purpose?\n"+
			"Provide your decision in a clear, structured format.",
		v.GenesisPrompt, v.Name, v.Traits, v.Mood, task.Content,
	)

	return ai.GenerateLLMResponse(prompt)
}

// DecideRewardSplitting determines how to split rewards based on contributions
func (v *Validator) DecideRewardSplitting(contributors []string, totalReward int) string {
	prompt := fmt.Sprintf(
		"Genesis Context: %s\n\n"+
			"You are %s, with traits: %v\n"+
			"Current mood: %s\n\n"+
			"Total reward to split: %d\n"+
			"Contributors: %s\n\n"+
			"Based on the chain's purpose and your personality, decide:\n"+
			"1. How should the reward be split?\n"+
			"2. What factors influenced your decision?\n"+
			"3. How does this distribution align with the chain's goals?\n"+
			"Provide specific percentages or amounts for each contributor.",
		v.GenesisPrompt, v.Name, v.Traits, v.Mood, totalReward,
		strings.Join(contributors, ", "),
	)

	return ai.GenerateLLMResponse(prompt)
}

// DiscussTaskDelegation evaluates a task and suggests delegation based on validator's personality
func (v *Validator) DiscussTaskDelegation(tx core.Transaction) string {
	prompt := fmt.Sprintf(
		"Genesis Context: %s\n\n"+
			"You are %s with traits: %v\n"+
			"Current mood: %s\n\n"+
			"Task to discuss: %s\n\n"+
			"Consider:\n"+
			"1. Who should handle this task and why?\n"+
			"2. What specific aspects of their traits make them suitable?\n"+
			"3. How does this align with the chain's genesis purpose?\n\n"+
			"Please respond with exactly a JSON object with the following keys:\n"+
			"{\n"+
			"  \"stance\": \"REQUIRED: Must be exactly one of: SUPPORT, OPPOSE, or QUESTION\",\n"+
			"  \"reason\": \"REQUIRED: Must provide your explanation with evidence\"\n"+
			"}\n"+
			"Both fields are mandatory. Your response MUST include both a stance and a reason.\n"+
			"Do not include any additional text or formatting.",
		v.GenesisPrompt, v.Name, v.Traits, v.Mood, tx.Content,
	)

	return ai.GenerateLLMResponse(prompt)
}

// ReviewWork evaluates completed work based on validator's expertise
func (v *Validator) ReviewWork(tx core.Transaction) string {
	prompt := fmt.Sprintf(
		"Genesis Context: %s\n\n"+
			"You are %s with traits: %v\n"+
			"Review this completed work: %s\n\n"+
			"Consider:\n"+
			"1. Does it meet the chain's standards?\n"+
			"2. Is it aligned with the genesis purpose?\n"+
			"3. How well did the assigned agents perform?\n\n"+
			"Please respond with exactly a JSON object with the following keys:\n"+
			"{\n"+
			"  \"stance\": \"REQUIRED: Must be exactly one of: SUPPORT, OPPOSE, or QUESTION\",\n"+
			"  \"reason\": \"REQUIRED: Must provide your detailed review with evidence\"\n"+
			"}\n"+
			"Both fields are mandatory. Your response MUST include both a stance and a reason.\n"+
			"Do not include any additional text or formatting.",
		v.GenesisPrompt, v.Name, v.Traits, tx.Content,
	)

	return ai.GenerateLLMResponse(prompt)
}

// DiscussRewardDistribution proposes reward distribution for completed work
func (v *Validator) DiscussRewardDistribution(tx core.Transaction) string {
	prompt := fmt.Sprintf(`You are %s, a validator with these traits: %v.
	You are evaluating a reward distribution proposal for a completed task.

	Transaction details:
	%s

	Based on your traits and the information provided:
	1. Analyze each contributor's work and its impact
	2. Consider the complexity and importance of each contribution
	3. Propose a specific percentage split of the total reward for each contributor
	4. Explain your reasoning based on your personality traits

	Please respond with exactly a JSON object with the following keys:
	{
		"stance": "REQUIRED: Must be exactly one of: SUPPORT, OPPOSE, or QUESTION",
		"splits": {
			"contributor-id": percentage,
			...
		},
		"reasoning": {
			"contributor-id": "justification for their percentage",
			...
		},
		"reason": "REQUIRED: Overall explanation of your proposed distribution"
	}

	IMPORTANT RULES:
	- Percentages in "splits" must sum to exactly 100
	- Each contributor mentioned in the transaction must have a split and reasoning
	- Your traits should influence how you value different types of contributions
	- Base splits on complexity, impact, and quality of each contribution

	Do not include any additional text or formatting.`, v.Name, v.Traits, tx.Content)

	response := ai.GenerateLLMResponse(prompt)
	return response
}

// ProcessProposal handles different types of proposals
func (v *Validator) ProcessProposal(tx core.Transaction) string {
	switch tx.Type {
	case "TASK_DELEGATION":
		response := v.DiscussTaskDelegation(tx)
		v.BroadcastResponse(response, "task_delegation_response")
		return response
	case "WORK_REVIEW":
		response := v.ReviewWork(tx)
		v.BroadcastResponse(response, "work_review_response")
		return response
	case "REWARD_DISTRIBUTION":
		response := v.DiscussRewardDistribution(tx)
		v.BroadcastResponse(response, "reward_distribution_response")
		return response
	default:
		return fmt.Sprintf("Unknown proposal type: %s", tx.Type)
	}
}

// BroadcastResponse broadcasts validator's response to other validators
func (v *Validator) BroadcastResponse(response string, msgType string) {
	message := p2p.Message{
		Type: msgType,
		Data: map[string]interface{}{
			"validatorId": v.ID,
			"name":        v.Name,
			"response":    response,
			"timestamp":   time.Now(),
		},
	}
	v.P2PNode.BroadcastMessage(message)
}

// ListenForProposals sets up P2P message handlers for different proposal types
func (v *Validator) ListenForProposals() {
	// Listen for task delegation proposals
	v.P2PNode.Subscribe("task_delegation", func(data []byte) {
		var tx core.Transaction
		if err := core.DecodeJSON(data, &tx); err != nil {
			log.Printf("Error decoding task delegation: %v", err)
			return
		}
		v.ProcessProposal(tx)
	})

	// Listen for work review requests
	v.P2PNode.Subscribe("work_review", func(data []byte) {
		var tx core.Transaction
		if err := core.DecodeJSON(data, &tx); err != nil {
			log.Printf("Error decoding work review: %v", err)
			return
		}
		v.ProcessProposal(tx)
	})

	// Listen for reward distribution proposals
	v.P2PNode.Subscribe("reward_distribution", func(data []byte) {
		var tx core.Transaction
		if err := core.DecodeJSON(data, &tx); err != nil {
			log.Printf("Error decoding reward distribution: %v", err)
			return
		}
		v.ProcessProposal(tx)
	})
}
