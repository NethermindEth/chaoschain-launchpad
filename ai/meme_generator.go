package ai

import (
	"strings"

	"github.com/NethermindEth/chaoschain-launchpad/core"
)

// GenerateMeme generates a meme response for block validation
func GenerateMeme(block core.Block, decision string) string {
	isValid := strings.Contains(decision, "VALID")
	if isValid {
		return "🎉 Much valid! Very block! Wow!"
	}
	return "😤 No block for you! Come back one year!"
}
