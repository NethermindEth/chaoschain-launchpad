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
