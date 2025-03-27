package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

type mockLLM struct{}

func (m *mockLLM) GenerateResponse(prompt string, context string, traits []string) string {
	// Generate deterministic responses based on the type of request
	if strings.Contains(prompt, "TASK_DELEGATION") {
		return generateTaskResponse(traits)
	} else if strings.Contains(prompt, "WORK_REVIEW") {
		return generateReviewResponse(traits)
	} else if strings.Contains(prompt, "REWARD_DISTRIBUTION") {
		return generateRewardResponse(traits)
	}
	return ""
}

func generateTaskResponse(traits []string) string {
	response := map[string]string{
		"stance": "SUPPORT",
		"reason": fmt.Sprintf("As someone focused on %s, I believe this task is well-defined and important. The authentication system requires careful consideration of security and user experience.", strings.Join(traits, " and ")),
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse)
}

func generateReviewResponse(traits []string) string {
	response := map[string]string{
		"stance": "SUPPORT",
		"reason": fmt.Sprintf("Based on my expertise in %s, the implementation meets our standards. The test coverage is good and documentation is comprehensive.", strings.Join(traits, " and ")),
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse)
}

func generateRewardResponse(traits []string) string {
	response := map[string]string{
		"stance": "SUPPORT",
		"reason": fmt.Sprintf("Considering my focus on %s, the proposed reward distribution fairly reflects the contribution and quality of work.", strings.Join(traits, " and ")),
	}
	jsonResponse, _ := json.Marshal(response)
	return string(jsonResponse)
}
