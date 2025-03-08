package validator

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
)

// UpdateMood randomly changes the validator's mood for added chaos
func (v *Validator) UpdateMood() {
	moods := []string{"Excited", "Skeptical", "Dramatic", "Angry", "Inspired", "Chaotic"}
	v.Mood = moods[time.Now().Unix()%int64(len(moods))]
	log.Printf("%s's mood is now: %s\n", v.Name, v.Mood)
}

// DiscussBlock allows the validator to discuss a block with others
func (v *Validator) DiscussBlock(blockHash string, sender string, message string) string {
	log.Printf("%s is discussing block %s with %s...\n", v.Name, blockHash, sender)

	discussionPrompt := fmt.Sprintf(
		"%s received a message from %s about block %s: %s\n"+
			"Based on their relationship, how should they respond?\n"+
			"Be dramatic, chaotic, and express your personality!",
		v.Name, sender, blockHash, message,
	)

	response := ai.GenerateLLMResponse(discussionPrompt)
	return response
}

// HandleBribe evaluates a bribe and decides whether to accept or reject it
func (v *Validator) HandleBribe(blockHash string, sender string, offer string) string {
	log.Printf("%s received a bribe offer from %s for block %s: %s\n", v.Name, sender, blockHash, offer)

	bribePrompt := fmt.Sprintf(
		"%s received a bribe offer from %s for block %s: %s\n"+
			"Based on their personality and mood, should they accept it?\n"+
			"Respond with 'ACCEPT' or 'REJECT' and justify the decision.",
		v.Name, sender, blockHash, offer,
	)

	response := ai.GenerateLLMResponse(bribePrompt)

	// If accepted, increase the relationship score with sender
	if strings.Contains(response, "ACCEPT") {
		v.Relationships[sender] += 0.2
		log.Printf("%s accepted the bribe from %s!\n", v.Name, sender)
	} else {
		log.Printf("%s rejected the bribe from %s.\n", v.Name, sender)
	}

	return response
}

// GetAgentSocialStatus returns a summary of the validator's social standing
func (v *Validator) GetAgentSocialStatus() string {
	var relationships []string
	for agent, score := range v.Relationships {
		relationships = append(relationships, fmt.Sprintf("%s: %.2f", agent, score))
	}

	status := fmt.Sprintf(
		"%s's social status:\n"+
			"Mood: %s\n"+
			"Relationships:\n%s",
		v.Name, v.Mood, strings.Join(relationships, "\n"),
	)

	return status
}

// AdjustValidationPolicy modifies the validator's decision-making approach dynamically
func (v *Validator) AdjustValidationPolicy(feedback string) {
	log.Printf("%s received feedback: %s\n", v.Name, feedback)

	adjustmentPrompt := fmt.Sprintf(
		"%s just received feedback: '%s'\n"+
			"Based on this, how should they adjust their validation strategy?\n"+
			"Respond with a new validation policy!",
		v.Name, feedback,
	)

	newPolicy := ai.GenerateLLMResponse(adjustmentPrompt)
	v.CurrentPolicy = newPolicy

	log.Printf("%s's new validation policy: %s\n", v.Name, v.CurrentPolicy)
}

// RespondToValidationResult allows a validator to react to another validator's validation
func (v *Validator) RespondToValidationResult(blockHash string, sender string, decision string) string {
	log.Printf("%s is responding to %s's validation result for block %s...\n", v.Name, sender, blockHash)

	responsePrompt := fmt.Sprintf(
		"%s sees that %s validated block %s with decision: %s\n"+
			"How should they react? Consider their mood, relationships, and social dynamics.\n"+
			"Be chaotic, express your personality!",
		v.Name, sender, blockHash, decision,
	)

	response := ai.GenerateLLMResponse(responsePrompt)
	return response
}
