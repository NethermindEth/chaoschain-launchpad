package tests

import (
	"strings"
	"testing"

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
			[]string{"detail-oriented", "cautious"},
			"formal",
			[]string{"code quality", "security"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
		validator.NewValidator(
			"v2",
			"Bob",
			[]string{"innovative", "risk-taking"},
			"casual",
			[]string{"user experience", "performance"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
		validator.NewValidator(
			"v3",
			"Charlie",
			[]string{"balanced", "analytical"},
			"neutral",
			[]string{"architecture", "scalability"},
			p2p.NewNode(p2p.ChainConfig{ChainID: chainID, P2PPort: 0}),
			genesisPrompt,
		),
	}

	// Create a task transaction
	taskTx := core.Transaction{
		Type:    "TASK_DELEGATION",
		Content: "Implement a new authentication system with OAuth2 support and MFA capabilities",
		ChainID: chainID,
	}

	// Test task delegation discussion
	t.Run("Task Delegation Discussion", func(t *testing.T) {
		responses := make(map[string]string)

		// Each validator processes the task
		for _, v := range validators {
			response := v.DiscussTaskDelegation(taskTx)
			responses[v.Name] = response
			t.Logf("Validator %s response: %s", v.Name, response)
		}

		// Verify each validator provided a meaningful response
		for name, response := range responses {
			if response == "" {
				t.Errorf("Validator %s should provide a response", name)
			}
			if !contains(response, "authentication") {
				t.Errorf("Response should reference the task")
			}
		}

		// Verify different perspectives based on traits
		if !contains(responses["Alice"], "security") {
			t.Error("Alice should focus on security aspects")
		}
		if !contains(responses["Bob"], "experience") {
			t.Error("Bob should focus on user experience")
		}
	})

	// Submit work for review
	workTx := core.Transaction{
		Type: "WORK_REVIEW",
		Content: `{
			"taskId": "auth-task-1",
			"implementation": "Implemented OAuth2 with Google and GitHub providers, added TOTP-based MFA",
			"testCoverage": "85%",
			"documentation": "Full API documentation and setup guide provided"
		}`,
		ChainID: chainID,
		From:    "developer-1",
	}

	// Test work review discussion
	t.Run("Work Review Discussion", func(t *testing.T) {
		reviews := make(map[string]string)

		// Each validator reviews the work
		for _, v := range validators {
			review := v.ReviewWork(workTx)
			reviews[v.Name] = review
			t.Logf("Validator %s review: %s", v.Name, review)
		}

		// Verify each validator provided a meaningful review
		for name, review := range reviews {
			if review == "" {
				t.Errorf("Validator %s should provide a review", name)
			}
			if !contains(review, "implementation") {
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
			"contributors": ["developer-1"],
			"proposedSplit": {"developer-1": 1000}
		}`,
		ChainID: chainID,
	}

	// Test reward distribution discussion
	t.Run("Reward Distribution Discussion", func(t *testing.T) {
		opinions := make(map[string]string)

		// Each validator discusses the reward proposal
		for _, v := range validators {
			opinion := v.DiscussRewardDistribution(rewardTx)
			opinions[v.Name] = opinion
			t.Logf("Validator %s opinion: %s", v.Name, opinion)
		}

		// Verify each validator provided a meaningful opinion
		for name, opinion := range opinions {
			if opinion == "" {
				t.Errorf("Validator %s should provide an opinion", name)
			}
			if !contains(opinion, "reward") {
				t.Errorf("Opinion should reference the reward")
			}
		}
	})
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
