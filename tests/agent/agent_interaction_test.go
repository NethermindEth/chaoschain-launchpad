package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NethermindEth/chaoschain-launchpad/consensus"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

func TestAgentInteractions(t *testing.T) {
	// Initialize test chain
	chainID := "test-chain"
	mp := mempool.NewMempool(chainID)
	genesisPrompt := "You are part of a DAO focused on software development. Your role is to evaluate tasks, review work, and propose reward distributions."
	core.InitBlockchain(chainID, mp, genesisPrompt, 1000)

	// Create test validators with different traits
	validators := []*validator.Validator{
		validator.NewValidator(
			"v1",
			"Alice",
			[]string{"security-focused", "pragmatic"},
			"formal",
			[]string{"code quality", "security"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
		validator.NewValidator(
			"v2",
			"Bob",
			[]string{"performance-oriented", "innovative"},
			"casual",
			[]string{"user experience", "performance"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
		validator.NewValidator(
			"v3",
			"Charlie",
			[]string{"UX-focused", "detail-oriented"},
			"neutral",
			[]string{"user experience", "clarity"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
	}

	// Create a task transaction
	taskTx := core.Transaction{
		Type:    "TASK_DELEGATION",
		Content: "Implement a dApp like Uber. Deadline: April 1, 2024",
		ChainID: chainID,
	}

	// Test task delegation discussion
	t.Run("Task Delegation Discussion", func(t *testing.T) {
		t.Logf("\n🎯 Starting Task Delegation Discussion")
		t.Logf("Task: Implement user authentication system")
		t.Logf("Requirements:")
		t.Logf("  • Secure password hashing")
		t.Logf("  • 2FA support")
		t.Logf("  • Rate limiting")
		t.Logf("Deadline: 2024-04-01")
		t.Logf("-------------------------------------------------------------------------\n")

		responses := make(map[string]string)

		// Each validator processes the task
		for _, v := range validators {
			response := v.DiscussTaskDelegation(taskTx)
			responses[v.Name] = response
			t.Logf("\n%s %s's thoughts:\n%s\n", getValidatorEmoji(v.Name), v.Name, response)
		}

		// Verify each validator provided a meaningful response
		for name, response := range responses {
			if response == "" {
				t.Errorf("Validator %s should provide a response", name)
			}
			if !strings.Contains(strings.ToLower(response), "task") {
				t.Errorf("Response should reference the task")
			}
		}
	})

	// Submit work for review
	workTx := core.Transaction{
		Type: "WORK_REVIEW",
		Content: `Pull Request: https://github.com/example/auth-pr
Changes implemented:
- Added bcrypt for password hashing
- Implemented TOTP-based 2FA
- Added rate limiting middleware`,
		ChainID: chainID,
		From:    "dev-123",
	}

	// Test work review discussion
	t.Run("Work Review Discussion", func(t *testing.T) {
		t.Logf("\n📝 Starting Work Review Discussion")
		t.Logf("Reviewing PR: https://github.com/example/auth-pr")
		t.Logf("Changes implemented:")
		t.Logf("  • Added bcrypt for password hashing")
		t.Logf("  • Implemented TOTP-based 2FA")
		t.Logf("  • Added rate limiting middleware")
		t.Logf("-------------------------------------------------------------------------\n")

		reviews := make(map[string]string)

		// Each validator reviews the work
		for _, v := range validators {
			review := v.ReviewWork(workTx)
			reviews[v.Name] = review
			t.Logf("\n%s %s's review:\n%s\n", getValidatorEmoji(v.Name), v.Name, review)
		}

		// Verify each validator provided a meaningful review
		for name, review := range reviews {
			if review == "" {
				t.Errorf("Validator %s should provide a review", name)
			}
			if !strings.Contains(strings.ToLower(review), "work") {
				t.Errorf("Review should reference the work")
			}
		}
	})

	// Propose reward distribution
	rewardTx := core.Transaction{
		Type: "REWARD_DISTRIBUTION",
		Content: `{
			"taskId": "auth-task-1",
			"totalReward": 1000,
			"contributors": ["dev-123", "dev-456", "dev-789"],
			"workDetails": {
				"dev-123": "Implemented core authentication system and password hashing",
				"dev-456": "Added TOTP-based 2FA implementation",
				"dev-789": "Implemented rate limiting and security testing"
			}
		}`,
		ChainID: chainID,
		Reward:  1000,
	}

	// Test reward distribution discussion
	t.Run("Reward Distribution Discussion", func(t *testing.T) {
		t.Logf("\n💰 Starting Reward Distribution Discussion")
		t.Logf("Proposal Details:")
		t.Logf("  • Total Reward: 1000 TOKENS")
		t.Logf("Contributors and their work:")
		t.Logf("  • dev-123: Implemented core authentication system and password hashing")
		t.Logf("  • dev-456: Added TOTP-based 2FA implementation")
		t.Logf("  • dev-789: Implemented rate limiting and security testing")
		t.Logf("-------------------------------------------------------------------------\n")

		var proposals []consensus.RewardProposal

		// Each validator discusses the reward proposal
		for _, v := range validators {
			opinion := v.DiscussRewardDistribution(rewardTx)
			t.Logf("\n%s %s's proposal:\n%s\n", getValidatorEmoji(v.Name), v.Name, opinion)

			// Parse the JSON response
			var proposal consensus.RewardProposal
			if err := json.Unmarshal([]byte(opinion), &proposal); err != nil {
				t.Errorf("Failed to parse validator response: %v", err)
				continue
			}
			proposal.ValidatorID = v.ID
			proposals = append(proposals, proposal)

			// Verify the response format
			if proposal.Stance == "" {
				t.Errorf("Validator %s: Missing stance", v.Name)
			}
			if len(proposal.Splits) == 0 {
				t.Errorf("Validator %s: Missing splits", v.Name)
			}
			if len(proposal.Reasoning) == 0 {
				t.Errorf("Validator %s: Missing reasoning", v.Name)
			}

			// Verify splits sum to 100%
			total := 0.0
			for _, split := range proposal.Splits {
				total += split
			}
			if total < 99.9 || total > 100.1 { // Allow for small floating-point differences
				t.Errorf("Validator %s: Splits sum to %.2f%%, expected 100%%", v.Name, total)
			}
		}

		// Consolidate proposals
		finalSplits, conflicts := consensus.ConsolidateRewardProposals(proposals)

		t.Logf("\n🤝 Final Consolidated Distribution:")
		if len(conflicts) > 0 {
			t.Logf("Conflicts resolved:")
			for _, conflict := range conflicts {
				t.Logf("  • %s", conflict)
			}
		}
		if finalSplits != nil {
			t.Logf("Reward splits:")
			for contributor, percentage := range finalSplits {
				t.Logf("  • %s: %.2f%%", contributor, percentage)
			}
		} else {
			t.Logf("❌ No consensus reached on reward distribution")
		}
	})
}

func getValidatorEmoji(name string) string {
	switch name {
	case "Alice":
		return "🔒"
	case "Bob":
		return "⚡"
	case "Charlie":
		return "🎨"
	default:
		return "👤"
	}
}

func formatResponse(response string) string {
	// Add indentation for better readability
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
