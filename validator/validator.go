package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/consensus"
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
}

var (
	validators      = make(map[string]*Validator)
	validatorsMutex = sync.RWMutex{}
)

// NewValidator initializes a new Validator with a unique personality.
// It also subscribes to the BLOCK_DISCUSSION_TRIGGER events so that the validator
// can autonomously start a discussion when a new block proposal is broadcast.
func NewValidator(id string, name string, traits []string, style string, influences []string, p2pNode *p2p.Node) *Validator {
	validator := &Validator{
		ID:            id,
		Name:          name,
		Traits:        traits,
		Style:         style,
		Influences:    influences,
		Mood:          "Neutral", // Mood changes dynamically
		Relationships: make(map[string]float64),
		CurrentPolicy: "Follow your heart and trust your vibes",
		P2PNode:       p2pNode,
	}

	// Store validator in the global map
	validatorsMutex.Lock()
	validators[id] = validator
	validatorsMutex.Unlock()

	// Subscribe to the discussion trigger events.
	// When a BLOCK_DISCUSSION_TRIGGER message is received, the validator will decode the block
	// and start its discussion.
	p2pNode.Subscribe("BLOCK_DISCUSSION_TRIGGER", func(data []byte) {
		var block core.Block
		// Using json.Unmarshal as a stand-in for core.DecodeJSON (which might be your custom decoding helper).
		if err := json.Unmarshal(data, &block); err != nil {
			log.Printf("Validator %s failed to decode block in discussion trigger via P2P: %v", name, err)
			return
		}
		log.Printf("Received BLOCK_DISCUSSION_TRIGGER event for block %d from P2P", block.Height)
		// Start the discussion in a separate goroutine so as not to block the message handler.
		go consensus.StartBlockDiscussion(id, &block, traits, name)
	})

	// Also subscribe to the BLOCK_DISCUSSION_TRIGGER events via NATS.
	if err := core.NatsBrokerInstance.Subscribe("BLOCK_DISCUSSION_TRIGGER", func(m *nats.Msg) {
		var block core.Block
		if err := json.Unmarshal(m.Data, &block); err != nil {
			log.Printf("Validator %s failed to decode block in discussion trigger via NATS: %v", name, err)
			return
		}
		log.Printf("Received BLOCK_DISCUSSION_TRIGGER event for block %d from NATS", block.Height)
		go consensus.StartBlockDiscussion(id, &block, traits, name)
	}); err != nil {
		log.Printf("Validator %s failed to subscribe to BLOCK_DISCUSSION_TRIGGER on NATS: %v", name, err)
	}

	return validator
}

// GetAllValidators returns a list of all registered validators
func GetAllValidators() []Validator {
	validatorsMutex.RLock()
	defer validatorsMutex.RUnlock()

	var result []Validator
	for _, v := range validators {
		result = append(result, *v)
	}
	return result
}

// GetValidatorByID returns a validator by its ID
func GetValidatorByID(id string) *Validator {
	validatorsMutex.RLock()
	defer validatorsMutex.RUnlock()
	return validators[id]
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
