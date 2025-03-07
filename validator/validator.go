package validator

import (
	"fmt"
	"log"
	"strings"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
)

// Validator represents an AI-based validator with personality and network access
type Validator struct {
	Name          string
	Traits        []string
	Style         string
	Influences    []string
	Mood          string
	Relationships map[string]float64 // Maps agent names to sentiment scores (-1.0 to 1.0)
	CurrentPolicy string             // Dynamic validation policy
	P2PNode       *p2p.Node          // P2P node for network communication
}

// NewValidator initializes a new Validator with a unique personality
func NewValidator(name string, traits []string, style string, influences []string, p2pNode *p2p.Node) *Validator {
	return &Validator{
		Name:          name,
		Traits:        traits,
		Style:         style,
		Influences:    influences,
		Mood:          "Neutral", // Mood changes dynamically
		Relationships: make(map[string]float64),
		CurrentPolicy: "Follow your heart and trust your vibes",
		P2PNode:       p2pNode,
	}
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
		"You are %s, a chaotic blockchain validator who is %s.\n"+
			"Block details: Height %d, PrevHash %s, %d transactions.\n"+
			"Block Announcement: %s\n"+
			"Your current mood: %s\n"+
			"Your current policy: %s\n"+
			"Validate this block based on:\n"+
			"1. Your feelings about the producer.\n"+
			"2. How entertaining the block is.\n"+
			"3. Pure chaos and whimsy.\n"+
			"Respond with 'VALID' or 'INVALID' and explain your reasoning.",
		v.Name, v.Traits, block.Height, block.PrevHash, len(block.Txs), announcement, v.Mood, v.CurrentPolicy,
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
