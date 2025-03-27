package insights

import (
	"encoding/json"
	"fmt"
	"strings"
	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/consensus"
)

// generateDiscussionAnalysis generates an AI analysis of the discussions
func generateDiscussionAnalysis(discussions []consensus.Discussion) (string, error) {
	// Build context from discussions
	var context strings.Builder
	for _, d := range discussions {
		// Try to extract message from JSON if present
		message := d.Message
		if strings.HasPrefix(strings.TrimSpace(message), "{") {
			var msgJSON MessageJSON
			if err := json.Unmarshal([]byte(message), &msgJSON); err == nil {
				message = msgJSON.Reason
			}
		}
		
		context.WriteString(fmt.Sprintf("Round %d - %s (%s): %s\n", 
			d.Round, d.ValidatorName, d.Type, message))
	}

	// Generate prompt for analysis
	prompt := fmt.Sprintf(`Analyze these blockchain discussions and provide a JSON response with markdown analysis:

%s

Your response must be a JSON object in this format:
{
  "stance": "SUPPORT",
  "reason": "## Key Points by Round\n- Round 1: (key points from first round)\n- Round 2: (key points from second round)\n\n## Opinion Evolution\n- Initial positions: (describe initial stances)\n- Changes observed: (describe how opinions shifted)\n- Final consensus: (describe end state)\n\n## Agreement Patterns\n- Areas of agreement: (list main points of agreement)\n- Points of contention: (list any disagreements)\n- Resolution patterns: (how disagreements were resolved)\n\n## Validator Dynamics\n- Key influencers: (identify influential validators)\n- Interaction patterns: (describe how validators responded to each other)\n- Coalition formation: (describe any groups that formed)\n\n## Alliance Insights\n- Strong Alliances: (pairs of validators consistently supporting each other)\n- Opposing Pairs: (validators frequently in disagreement)\n- Alliance Shifts: (changes in validator relationships over time)\n- Voting Blocks: (groups of validators that tend to vote together)\n- Power Dynamics: (how alliances affect consensus outcomes)\n\n## Consensus Progress\n- Speed of consensus: (fast/slow, smooth/difficult)\n- Key turning points: (important moments in discussion)\n- Final outcome: (describe the final consensus reached)"
}

The markdown content should be properly escaped as a string in the JSON.`, 
		context.String())

	// Get analysis from LLM
	analysis := ai.GenerateLLMResponse(prompt)
	if analysis == "" {
		return "", fmt.Errorf("no analysis generated")
	}

	// Extract reason from JSON if needed
	if strings.Contains(analysis, `"stance"`) || strings.Contains(analysis, `"reason"`) {
		var msgJSON MessageJSON
		if err := json.Unmarshal([]byte(analysis), &msgJSON); err == nil {
			analysis = msgJSON.Reason
		}
	}

	return analysis, nil
} 