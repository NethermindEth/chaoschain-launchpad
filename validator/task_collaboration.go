package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/core"
)

// TaskBreakdownRound represents a single round of task breakdown discussion
type TaskBreakdownRound struct {
	Round     int
	Proposals map[string]TaskBreakdownProposal // validatorID -> proposal
}

// TaskBreakdownProposal represents a validator's proposed task breakdown
type TaskBreakdownProposal struct {
	ValidatorID   string   `json:"validatorId"`
	ValidatorName string   `json:"validatorName"`
	Subtasks      []string `json:"subtasks"`
	Reasoning     string   `json:"reasoning"`
	Timestamp     time.Time
}

// TaskBreakdownResults contains the final consolidated task breakdown
type TaskBreakdownResults struct {
	FinalSubtasks      []string             // The final, agreed-upon list of subtasks
	DiscussionHistory  []TaskBreakdownRound // History of all discussion rounds
	ValidatorVotes     map[string][]string  // validatorID -> subtasks they supported
	BlockInfo          *core.Block          // The block that triggered this breakdown
	TransactionDetails string               // String representation of transaction details
}

// TaskDelegationRound represents a single round of task delegation discussion
type TaskDelegationRound struct {
	Round     int
	Proposals map[string]TaskDelegationProposal // validatorID -> proposal
}

// TaskDelegationProposal represents a validator's proposed task delegation
type TaskDelegationProposal struct {
	ValidatorID   string            `json:"validatorId"`
	ValidatorName string            `json:"validatorName"`
	Assignments   map[string]string `json:"assignments"` // subtask -> validator name
	Reasoning     string            `json:"reasoning"`
	Timestamp     time.Time
}

// TaskDelegationResults contains the final consolidated task delegations
type TaskDelegationResults struct {
	Assignments       map[string]string            // subtask -> validator name
	DiscussionHistory []TaskDelegationRound        // History of all discussion rounds
	ValidatorVotes    map[string]map[string]string // validatorID -> (subtask -> proposed validator)
	BlockInfo         *core.Block                  // The block that triggered this delegation
	Subtasks          []string                     // The subtasks being delegated
}

// AgentFeedback represents feedback from an agent on a proposal
type AgentFeedback struct {
	ValidatorID     string   `json:"validatorId"`
	ValidatorName   string   `json:"validatorName"`
	FeedbackType    string   `json:"feedbackType"`              // "support", "critique", "refine"
	Message         string   `json:"message"`                   // Detailed feedback message
	RefinedSubtasks []string `json:"refinedSubtasks,omitempty"` // Only present for "refine" type
	Timestamp       time.Time
}

// DecisionStrategy represents an agent's strategy for final decision making
type DecisionStrategy struct {
	ValidatorID   string `json:"validatorId"`
	ValidatorName string `json:"validatorName"`
	Strategy      string `json:"strategy"` // e.g., "consensus", "majority", "expert", etc.
	Reasoning     string `json:"reasoning"`
	Timestamp     time.Time
}

const (
	InitialProposalRound = 1
	FeedbackRound        = 2
	FinalizationRound    = 3
	RoundDuration        = 5 * time.Second // Time per round
)

var (
	taskBreakdownMutex  sync.Mutex
	taskDelegationMutex sync.Mutex
)

// StartCollaborativeTaskBreakdown initiates a multi-round task breakdown process among validators
func StartCollaborativeTaskBreakdown(chainID string, block *core.Block, transactionDetails string) *TaskBreakdownResults {
	validators := GetAllValidators(chainID)
	if len(validators) == 0 {
		log.Printf("No validators available for task breakdown")
		return nil
	}

	// Log transaction details that are being processed
	log.Printf("======= STARTING TASK BREAKDOWN =======")
	log.Printf("Block Height: %d", block.Height)
	log.Printf("Block Hash: %s", block.Hash())
	log.Printf("Transaction Details:")
	log.Printf("%s", transactionDetails)
	log.Printf("Number of Validators: %d", len(validators))
	for _, v := range validators {
		log.Printf("  - %s (%s)", v.Name, v.ID)
	}
	log.Printf("=======================================")

	// Initialize results structure
	results := &TaskBreakdownResults{
		DiscussionHistory:  make([]TaskBreakdownRound, 3), // 3 rounds: initial, feedback, finalization
		ValidatorVotes:     make(map[string][]string),
		BlockInfo:          block,
		TransactionDetails: transactionDetails,
	}

	// ROUND 1: Initial Proposals
	// Each validator presents their initial proposal and reasoning
	log.Printf("Starting Round 1: Initial Proposals")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskBreakdownRoundStart, map[string]interface{}{
		"round":       1,
		"blockHeight": block.Height,
		"timestamp":   time.Now(),
	})

	round1Proposals := make(map[string]TaskBreakdownProposal)
	var round1Wg sync.WaitGroup

	for _, validator := range validators {
		round1Wg.Add(1)
		go func(v *Validator) {
			defer round1Wg.Done()

			proposal := generateInitialProposal(v, results)

			taskBreakdownMutex.Lock()
			round1Proposals[v.ID] = proposal
			results.ValidatorVotes[v.ID] = proposal.Subtasks
			taskBreakdownMutex.Unlock()

			// Enhanced logging of proposal details
			log.Printf("\n📌 BREAKDOWN PROPOSAL (Round 1) from %s:", v.Name)
			log.Printf("  Subtasks proposed (%d):", len(proposal.Subtasks))
			for i, subtask := range proposal.Subtasks {
				log.Printf("  %d. %s", i+1, subtask)
			}
			log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
			log.Printf("  -----------------------------")

			// Broadcast for UI
			communication.BroadcastEvent(communication.EventTaskBreakdown, map[string]interface{}{
				"validatorId":   proposal.ValidatorID,
				"validatorName": proposal.ValidatorName,
				"subtasks":      proposal.Subtasks,
				"reasoning":     proposal.Reasoning,
				"round":         1,
				"blockHeight":   block.Height,
				"timestamp":     time.Now(),
			})

			log.Printf("Validator %s submitted initial proposal with %d subtasks",
				v.Name, len(proposal.Subtasks))
		}(validator)
	}

	round1Wg.Wait()
	results.DiscussionHistory[0] = TaskBreakdownRound{
		Round:     1,
		Proposals: round1Proposals,
	}
	log.Printf("Completed Round 1 with %d proposals", len(round1Proposals))

	// Wait between rounds
	time.Sleep(RoundDuration)

	// ROUND 2: Review, Critique, Support, or Refine
	// Agents review other proposals and provide feedback
	log.Printf("Starting Round 2: Feedback and Refinement")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskBreakdownRoundStart, map[string]interface{}{
		"round":       2,
		"blockHeight": block.Height,
		"timestamp":   time.Now(),
	})

	round2Proposals := make(map[string]TaskBreakdownProposal)
	var round2Wg sync.WaitGroup

	// Format round 1 proposals for context
	round1Context := formatProposalsForReview(round1Proposals)

	for _, validator := range validators {
		round2Wg.Add(1)
		go func(v *Validator) {
			defer round2Wg.Done()

			proposal := generateFeedbackProposal(v, round1Context, results)

			taskBreakdownMutex.Lock()
			round2Proposals[v.ID] = proposal
			results.ValidatorVotes[v.ID] = proposal.Subtasks
			taskBreakdownMutex.Unlock()

			// Enhanced logging of proposal details
			log.Printf("\n📝 BREAKDOWN FEEDBACK (Round 2) from %s:", v.Name)
			log.Printf("  Refined subtasks (%d):", len(proposal.Subtasks))
			for i, subtask := range proposal.Subtasks {
				log.Printf("  %d. %s", i+1, subtask)
			}
			log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
			log.Printf("  -----------------------------")

			// Broadcast for UI
			communication.BroadcastEvent(communication.EventTaskBreakdown, map[string]interface{}{
				"validatorId":   proposal.ValidatorID,
				"validatorName": proposal.ValidatorName,
				"subtasks":      proposal.Subtasks,
				"reasoning":     proposal.Reasoning,
				"round":         2,
				"blockHeight":   block.Height,
				"timestamp":     time.Now(),
			})

			log.Printf("Validator %s submitted feedback with %d subtasks",
				v.Name, len(proposal.Subtasks))
		}(validator)
	}

	round2Wg.Wait()
	results.DiscussionHistory[1] = TaskBreakdownRound{
		Round:     2,
		Proposals: round2Proposals,
	}
	log.Printf("Completed Round 2 with %d feedback proposals", len(round2Proposals))

	// Wait between rounds
	time.Sleep(RoundDuration)

	// ROUND 3: Final Decision
	// Agents continue discussions until they reach consensus
	log.Printf("Starting Round 3: Continuous Discussion Until Consensus")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskBreakdownRoundStart, map[string]interface{}{
		"round":       3,
		"blockHeight": block.Height,
		"timestamp":   time.Now(),
	})

	// Define consensus parameters
	maxIterations := 5
	consensusThreshold := 0.75 // At least 75% consensus needed

	// Store all iterations of proposals
	var allRound3Proposals []map[string]TaskBreakdownProposal
	var currentRound3Proposals map[string]TaskBreakdownProposal
	var consensusReached bool
	var iteration int
	var finalSubtasks []string

	// Initial discussion context is from rounds 1 and 2
	discussionContext := formatDiscussionHistory(results)

	// Loop until consensus reached or max iterations
	for iteration = 0; iteration < maxIterations && !consensusReached; iteration++ {
		log.Printf("Starting discussion iteration %d", iteration+1)

		currentRound3Proposals = make(map[string]TaskBreakdownProposal)
		var iterationWg sync.WaitGroup

		// Current iteration context includes all previous round 3 discussions
		currentContext := discussionContext
		if iteration > 0 {
			// Add previous round 3 discussions to context
			currentContext += "\n\nPREVIOUS DISCUSSION ATTEMPTS:\n\n"
			for i, prevRoundProposals := range allRound3Proposals {
				currentContext += fmt.Sprintf("ITERATION %d:\n", i+1)
				currentContext += formatProposalsForReview(prevRoundProposals)
				currentContext += "\n"
			}
		}

		// Each validator submits a proposal
		for _, validator := range validators {
			iterationWg.Add(1)
			go func(v *Validator) {
				defer iterationWg.Done()

				var proposal TaskBreakdownProposal
				if iteration == 0 {
					// First iteration uses standard final decision function
					proposal = generateFinalDecision(v, currentContext, results)
				} else {
					// Subsequent iterations use consensus-building function
					proposal = generateConsensusProposal(v, currentContext, results, iteration)
				}

				taskBreakdownMutex.Lock()
				currentRound3Proposals[v.ID] = proposal
				results.ValidatorVotes[v.ID] = proposal.Subtasks
				taskBreakdownMutex.Unlock()

				// Enhanced logging of proposal details
				log.Printf("\n🧩 BREAKDOWN CONSENSUS (Round 3, Iteration %d) from %s:", iteration+1, v.Name)
				log.Printf("  Proposed subtasks (%d):", len(proposal.Subtasks))
				for i, subtask := range proposal.Subtasks {
					log.Printf("  %d. %s", i+1, subtask)
				}
				log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
				log.Printf("  -----------------------------")

				// Broadcast for UI
				communication.BroadcastEvent(communication.EventTaskBreakdown, map[string]interface{}{
					"validatorId":   proposal.ValidatorID,
					"validatorName": proposal.ValidatorName,
					"subtasks":      proposal.Subtasks,
					"reasoning":     proposal.Reasoning,
					"round":         3,
					"iteration":     iteration + 1,
					"blockHeight":   block.Height,
					"timestamp":     time.Now(),
				})

				log.Printf("Validator %s submitted consensus proposal %d with %d subtasks",
					v.Name, iteration+1, len(proposal.Subtasks))
			}(validator)
		}

		iterationWg.Wait()
		allRound3Proposals = append(allRound3Proposals, currentRound3Proposals)

		// Check for consensus
		finalSubtasks = consolidateFinalDecisions(currentRound3Proposals)
		consensusScore := calculateConsensusScore(currentRound3Proposals, finalSubtasks)

		log.Printf("Consensus iteration %d complete - consensus score: %.2f (threshold: %.2f)",
			iteration+1, consensusScore, consensusThreshold)

		// Broadcast iteration result
		communication.BroadcastEvent(communication.EventTaskBreakdownRoundIteration, map[string]interface{}{
			"round":            3,
			"iteration":        iteration + 1,
			"consensusScore":   consensusScore,
			"threshold":        consensusThreshold,
			"consensusReached": consensusScore >= consensusThreshold,
			"blockHeight":      block.Height,
			"timestamp":        time.Now(),
		})

		if consensusScore >= consensusThreshold {
			consensusReached = true
			log.Printf("Consensus reached after %d iterations!", iteration+1)

			// Log detailed final breakdown consensus
			log.Printf("\n====== FINAL TASK BREAKDOWN CONSENSUS DETAILS ======")
			log.Printf("Consensus Score: %.2f (Threshold: %.2f)", consensusScore, consensusThreshold)
			log.Printf("Iterations Required: %d of %d maximum", iteration+1, maxIterations)
			log.Printf("\nFinal agreed subtasks (%d):", len(finalSubtasks))
			for i, subtask := range finalSubtasks {
				log.Printf("%d. %s", i+1, subtask)
			}

			log.Printf("\nValidator Contributions:")
			for _, proposal := range currentRound3Proposals {
				numMatches := 0
				for _, consensusTask := range finalSubtasks {
					for _, proposedTask := range proposal.Subtasks {
						if strings.TrimSpace(proposedTask) == strings.TrimSpace(consensusTask) {
							numMatches++
							break
						}
					}
				}

				// Calculate match percentage
				matchPercentage := 0.0
				if len(finalSubtasks) > 0 {
					matchPercentage = float64(numMatches) / float64(len(finalSubtasks)) * 100
				}

				log.Printf("\n🧠 %s's contribution:", proposal.ValidatorName)
				log.Printf("  Consensus: %.1f%% (%d of %d subtasks)",
					matchPercentage, numMatches, len(finalSubtasks))
				log.Printf("  Unique contributions: %d", len(proposal.Subtasks)-numMatches)
				log.Printf("  Full reasoning:")
				log.Printf("  %s", proposal.Reasoning)
			}
			log.Printf("\n================================================")
		} else {
			// Wait between iterations
			time.Sleep(RoundDuration / 2)
		}
	}

	// Store the final round results
	results.DiscussionHistory[2] = TaskBreakdownRound{
		Round:     3,
		Proposals: currentRound3Proposals,
	}

	if !consensusReached {
		log.Printf("WARNING: Max iterations (%d) reached without sufficient consensus. Using best available list.", maxIterations)
	}

	// If no subtasks were found, create some generic ones
	if len(finalSubtasks) == 0 {
		log.Printf("WARNING: No subtasks were found in the final decisions. Using generic subtasks.")
		finalSubtasks = []string{
			"Research requirements and existing solutions",
			"Design system architecture",
			"Implement core functionality",
			"Test the implementation",
			"Deploy and document the solution",
		}
	}

	results.FinalSubtasks = finalSubtasks
	log.Printf("Task breakdown completed with %d subtasks", len(finalSubtasks))

	// Add comprehensive summary information
	log.Printf("\n======= TASK BREAKDOWN SUMMARY =======")
	log.Printf("Process completed at: %s", time.Now().Format(time.RFC3339))
	log.Printf("Block Height: %d, Hash: %s", results.BlockInfo.Height, results.BlockInfo.Hash())
	log.Printf("Sufficient consensus achieved: %v (Score: %.2f)", consensusReached, calculateConsensusScore(currentRound3Proposals, finalSubtasks))
	log.Printf("Rounds completed: %d standard + %d discussion iterations", 2, iteration)
	log.Printf("Validators participating: %d", len(validators))

	// Log proposal statistics
	var totalProposals, totalSubtasksMentioned int
	uniqueSubtasks := make(map[string]int)

	// Round 1
	for _, proposal := range results.DiscussionHistory[0].Proposals {
		totalProposals++
		for _, subtask := range proposal.Subtasks {
			totalSubtasksMentioned++
			uniqueSubtasks[strings.TrimSpace(subtask)]++
		}
	}

	// Round 2
	for _, proposal := range results.DiscussionHistory[1].Proposals {
		totalProposals++
		for _, subtask := range proposal.Subtasks {
			totalSubtasksMentioned++
			uniqueSubtasks[strings.TrimSpace(subtask)]++
		}
	}

	// Round 3 (all iterations)
	for _, iterProposals := range allRound3Proposals {
		for _, proposal := range iterProposals {
			totalProposals++
			for _, subtask := range proposal.Subtasks {
				totalSubtasksMentioned++
				uniqueSubtasks[strings.TrimSpace(subtask)]++
			}
		}
	}

	log.Printf("Total proposals generated: %d", totalProposals)
	log.Printf("Total subtasks mentioned: %d", totalSubtasksMentioned)
	log.Printf("Unique subtasks proposed: %d", len(uniqueSubtasks))
	log.Printf("Final subtasks selected: %d", len(finalSubtasks))

	// Top mentioned subtasks
	type SubtaskCount struct {
		Subtask string
		Count   int
	}

	var subtaskCounts []SubtaskCount
	for subtask, count := range uniqueSubtasks {
		subtaskCounts = append(subtaskCounts, SubtaskCount{subtask, count})
	}

	// Sort by count
	sort.Slice(subtaskCounts, func(i, j int) bool {
		return subtaskCounts[i].Count > subtaskCounts[j].Count
	})

	// Show top mentioned subtasks
	log.Printf("\nTop mentioned subtasks:")
	for i, sc := range subtaskCounts {
		if i >= 5 {
			break
		}
		log.Printf("%d. \"%s\" (mentioned %d times)", i+1, sc.Subtask, sc.Count)
	}

	log.Printf("\nFinal subtasks selected:")
	for i, subtask := range finalSubtasks {
		count := uniqueSubtasks[strings.TrimSpace(subtask)]
		log.Printf("%d. \"%s\" (mentioned %d times)", i+1, subtask, count)
	}

	log.Printf("=======================================")

	// Broadcast final breakdown
	communication.BroadcastEvent(communication.EventTaskBreakdownFinal, map[string]interface{}{
		"subtasks":         finalSubtasks,
		"blockHeight":      block.Height,
		"consensusReached": consensusReached,
		"iterationsNeeded": iteration,
		"timestamp":        time.Now(),
	})

	return results
}

// generateInitialProposal creates an initial task breakdown proposal from a validator
func generateInitialProposal(v *Validator, results *TaskBreakdownResults) TaskBreakdownProposal {
	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 1 (Initial Proposal) of a collaborative task breakdown process.

The following task needs to be broken down:
%s

Block Information:
- Height: %d
- Hash: %s
- Proposer: %s
- Timestamp: %d

Your task is to provide an INITIAL BREAKDOWN of this request into clear, manageable subtasks.
Focus on creating a comprehensive, logical breakdown that addresses all aspects of the task.

Please respond with a JSON object containing:
{
  "subtasks": ["Subtask 1 description", "Subtask 2 description", ...],
  "reasoning": "Your explanation of why you chose this breakdown and your approach to analyzing the task"
}

Ensure your subtasks are clear, specific, and implementable. Your reasoning should explain your thought process.`,
		v.Name, v.Traits, results.TransactionDetails,
		results.BlockInfo.Height, results.BlockInfo.Hash(),
		results.BlockInfo.Proposer, results.BlockInfo.Timestamp)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var proposalData struct {
		Subtasks  []string `json:"subtasks"`
		Reasoning string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &proposalData); err != nil {
		log.Printf("Error parsing initial task breakdown proposal from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		proposalData.Subtasks = []string{"Error parsing response"}
		proposalData.Reasoning = "Error parsing AI response"
	}

	return TaskBreakdownProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Subtasks:      proposalData.Subtasks,
		Reasoning:     proposalData.Reasoning,
		Timestamp:     time.Now(),
	}
}

// generateFeedbackProposal creates a proposal with feedback on other proposals
func generateFeedbackProposal(v *Validator, proposalsContext string, results *TaskBreakdownResults) TaskBreakdownProposal {
	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 2 (Feedback) of a collaborative task breakdown process.

Original Task:
%s

INITIAL PROPOSALS from validators:
%s

Your task is to REVIEW the initial proposals from other validators, then:
1. CRITIQUE what's missing or could be improved
2. SUPPORT aspects you think are strong
3. REFINE the proposals into a better task breakdown

Based on your traits and expertise, provide your perspective on how the task should be broken down.

Please respond with a JSON object containing:
{
  "feedback": "Your critique and/or support for other proposals",
  "subtasks": ["Your refined subtask 1", "Your refined subtask 2", ...],
  "reasoning": "Explanation of your refinements and how they improve upon the initial proposals"
}

Be specific in your feedback and create a subtask list that addresses any issues you identified.`,
		v.Name, v.Traits, results.TransactionDetails, proposalsContext)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var feedbackData struct {
		Feedback  string   `json:"feedback"`
		Subtasks  []string `json:"subtasks"`
		Reasoning string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &feedbackData); err != nil {
		log.Printf("Error parsing feedback proposal from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		feedbackData.Feedback = "Error parsing response"
		feedbackData.Subtasks = []string{"Error parsing response"}
		feedbackData.Reasoning = "Error parsing AI response"
	}

	// Combine feedback and reasoning
	combinedReasoning := fmt.Sprintf("Feedback on proposals:\n%s\n\nReasoning for refinements:\n%s",
		feedbackData.Feedback, feedbackData.Reasoning)

	return TaskBreakdownProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Subtasks:      feedbackData.Subtasks,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// generateFinalDecision creates a final decision proposal based on all previous discussion
func generateFinalDecision(v *Validator, discussionContext string, results *TaskBreakdownResults) TaskBreakdownProposal {
	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 3 (Final Decision) of a collaborative task breakdown process.

Original Task:
%s

DISCUSSION HISTORY (Initial Proposals and Feedback):
%s

Your task is to make a FINAL DECISION on the task breakdown.
Use a consensus-building approach that aims to incorporate the most valuable aspects of all proposals.
Focus on identifying common patterns and themes across different validators' proposals.

When creating your final subtask list, prioritize:
- Subtasks that appeared in multiple proposals (indicating broader consensus)
- Critical components that must be included even if only proposed by one validator
- A balanced approach that reflects the collective wisdom of the group

Please respond with a JSON object containing:
{
  "consensusStrategy": "Detailed description of how you're finding consensus among the proposals",
  "subtasks": ["Final subtask 1", "Final subtask 2", ...],
  "reasoning": "Explanation of why this final breakdown represents a good consensus"
}

Your subtasks should represent the best consensus that can be achieved based on the discussion so far.`,
		v.Name, v.Traits, results.TransactionDetails, discussionContext)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var decisionData struct {
		ConsensusStrategy string   `json:"consensusStrategy"`
		Subtasks          []string `json:"subtasks"`
		Reasoning         string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &decisionData); err != nil {
		log.Printf("Error parsing final decision from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		decisionData.ConsensusStrategy = "Error parsing response"
		decisionData.Subtasks = []string{"Error parsing response"}
		decisionData.Reasoning = "Error parsing AI response"
	}

	// Combine strategy and reasoning
	combinedReasoning := fmt.Sprintf("Consensus Strategy: %s\n\nReasoning:\n%s",
		decisionData.ConsensusStrategy, decisionData.Reasoning)

	return TaskBreakdownProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Subtasks:      decisionData.Subtasks,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// generateConsensusProposal creates a proposal for subsequent iterations aimed at building consensus
func generateConsensusProposal(v *Validator, discussionContext string, results *TaskBreakdownResults, iteration int) TaskBreakdownProposal {
	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in an EXTENDED Round 3 (Consensus Building) of a collaborative task breakdown process.
This is iteration %d of the consensus-building process.

Original Task:
%s

COMPLETE DISCUSSION HISTORY (including previous consensus attempts):
%s

Your task is to FIND CONSENSUS with the other validators.
Review all previous proposals, especially the most recent iteration, and look for common ground.
Focus on refining and merging popular ideas rather than introducing entirely new concepts at this stage.

Please respond with a JSON object containing:
{
  "consensusStrategy": "Explain how you're trying to bridge gaps between different proposals to reach consensus",
  "subtasks": ["Final subtask 1", "Final subtask 2", ...],
  "reasoning": "Explain why this list represents a good consensus that addresses the most important points from multiple validators"
}

Your goal is to help the group reach consensus, not to push your own preferences.
Identify which subtasks have broader support and adapt your proposal accordingly.`,
		v.Name, v.Traits, iteration+1, results.TransactionDetails, discussionContext)

	// Log the consensus-building prompt
	log.Printf("\n🔄 CONSENSUS PROMPT for %s (Iteration %d):\n%s\n", v.Name, iteration+1, prompt)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var consensusData struct {
		ConsensusStrategy string   `json:"consensusStrategy"`
		Subtasks          []string `json:"subtasks"`
		Reasoning         string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &consensusData); err != nil {
		log.Printf("Error parsing consensus proposal from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		consensusData.ConsensusStrategy = "Error parsing response"
		consensusData.Subtasks = []string{"Error parsing response"}
		consensusData.Reasoning = "Error parsing AI response"
	}

	// Combine strategy and reasoning
	combinedReasoning := fmt.Sprintf("Consensus Strategy (Iteration %d): %s\n\nReasoning:\n%s",
		iteration+1, consensusData.ConsensusStrategy, consensusData.Reasoning)

	return TaskBreakdownProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Subtasks:      consensusData.Subtasks,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// formatProposalsForReview formats proposals for review by other validators
func formatProposalsForReview(proposals map[string]TaskBreakdownProposal) string {
	var result strings.Builder

	for _, proposal := range proposals {
		result.WriteString(fmt.Sprintf("Validator: %s\n", proposal.ValidatorName))
		result.WriteString("Subtasks:\n")

		for i, subtask := range proposal.Subtasks {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, subtask))
		}

		result.WriteString(fmt.Sprintf("Reasoning: %s\n\n", proposal.Reasoning))
	}

	return result.String()
}

// formatDiscussionHistory formats the entire discussion history for the final round
func formatDiscussionHistory(results *TaskBreakdownResults) string {
	var result strings.Builder

	// Round 1: Initial Proposals
	result.WriteString("ROUND 1 - INITIAL PROPOSALS:\n\n")
	result.WriteString(formatProposalsForReview(results.DiscussionHistory[0].Proposals))

	// Round 2: Feedback
	result.WriteString("\nROUND 2 - FEEDBACK AND REFINEMENTS:\n\n")
	result.WriteString(formatProposalsForReview(results.DiscussionHistory[1].Proposals))

	return result.String()
}

// consolidateFinalDecisions analyzes final decisions and extracts the most agreed-upon subtasks
func consolidateFinalDecisions(finalProposals map[string]TaskBreakdownProposal) []string {
	// Count how many validators included each subtask in their final list
	subtaskCounts := make(map[string]int)

	// First, normalize and count all subtasks
	for _, proposal := range finalProposals {
		for _, subtask := range proposal.Subtasks {
			// Clean the subtask for comparison
			cleanSubtask := strings.TrimSpace(subtask)
			subtaskCounts[cleanSubtask]++
		}
	}

	// Create a slice of subtasks with their counts for sorting
	type SubtaskCount struct {
		Subtask string
		Count   int
	}

	var subtaskCountList []SubtaskCount
	for subtask, count := range subtaskCounts {
		subtaskCountList = append(subtaskCountList, SubtaskCount{subtask, count})
	}

	// Sort by count (descending)
	sort.Slice(subtaskCountList, func(i, j int) bool {
		return subtaskCountList[i].Count > subtaskCountList[j].Count
	})

	// Take the top N subtasks or those with at least 2 votes
	minVotes := 1
	if len(finalProposals) >= 3 {
		minVotes = 2
	}

	var finalSubtasks []string
	for _, sc := range subtaskCountList {
		if sc.Count >= minVotes {
			finalSubtasks = append(finalSubtasks, sc.Subtask)
		}
	}

	// If we have too few subtasks, take the top 5
	if len(finalSubtasks) < 3 && len(subtaskCountList) > 0 {
		finalSubtasks = []string{}
		for i := 0; i < min(5, len(subtaskCountList)); i++ {
			finalSubtasks = append(finalSubtasks, subtaskCountList[i].Subtask)
		}
	}

	log.Printf("Extracted %d final subtasks from %d finalization proposals",
		len(finalSubtasks), len(finalProposals))

	return finalSubtasks
}

// calculateConsensusScore measures how much consensus exists across validators' proposals
// Returns a value between 0 (no consensus) and 1 (perfect consensus)
func calculateConsensusScore(proposals map[string]TaskBreakdownProposal, consensusSubtasks []string) float64 {
	if len(proposals) == 0 || len(consensusSubtasks) == 0 {
		return 0.0
	}

	// For each validator, calculate what percentage of the consensus subtasks they included
	var totalConsensusScore float64

	for _, proposal := range proposals {
		// Create a map of the validator's subtasks for O(1) lookup
		validatorSubtasks := make(map[string]bool)
		for _, subtask := range proposal.Subtasks {
			validatorSubtasks[strings.TrimSpace(subtask)] = true
		}

		// Count how many consensus subtasks this validator included
		var matches float64
		for _, consensusSubtask := range consensusSubtasks {
			if validatorSubtasks[strings.TrimSpace(consensusSubtask)] {
				matches++
			}
		}

		// Calculate consensus as percentage of consensus subtasks included
		consensusScore := matches / float64(len(consensusSubtasks))
		totalConsensusScore += consensusScore
	}

	// Average consensus across all validators
	return totalConsensusScore / float64(len(proposals))
}

// StartCollaborativeTaskDelegation initiates a multi-round task delegation process
func StartCollaborativeTaskDelegation(chainID string, taskBreakdown *TaskBreakdownResults) *TaskDelegationResults {
	validators := GetAllValidators(chainID)
	if len(validators) == 0 {
		log.Printf("No validators available for task delegation")
		return nil
	}

	// Log task breakdown details that we'll be delegating
	log.Printf("======= STARTING TASK DELEGATION =======")
	log.Printf("Block Height: %d", taskBreakdown.BlockInfo.Height)
	log.Printf("Block Hash: %s", taskBreakdown.BlockInfo.Hash())
	log.Printf("Subtasks to delegate:")
	for i, subtask := range taskBreakdown.FinalSubtasks {
		log.Printf("  %d. %s", i+1, subtask)
	}
	log.Printf("Number of Validators: %d", len(validators))
	for _, v := range validators {
		log.Printf("  - %s (%s)", v.Name, v.ID)
	}
	log.Printf("=======================================")

	// Initialize results structure
	results := &TaskDelegationResults{
		DiscussionHistory: make([]TaskDelegationRound, 3), // 3 rounds, like task breakdown
		ValidatorVotes:    make(map[string]map[string]string),
		BlockInfo:         taskBreakdown.BlockInfo,
		Subtasks:          taskBreakdown.FinalSubtasks,
	}

	// ROUND 1: Initial Delegation Proposals
	// Each validator presents their initial delegation proposal
	log.Printf("Starting Round 1: Initial Delegation Proposals")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskDelegationRoundStart, map[string]interface{}{
		"round":       1,
		"blockHeight": results.BlockInfo.Height,
		"timestamp":   time.Now(),
	})

	round1Proposals := make(map[string]TaskDelegationProposal)
	var round1Wg sync.WaitGroup

	for _, validator := range validators {
		round1Wg.Add(1)
		go func(v *Validator) {
			defer round1Wg.Done()

			proposal := generateInitialDelegation(v, results, validators)

			taskDelegationMutex.Lock()
			round1Proposals[v.ID] = proposal
			results.ValidatorVotes[v.ID] = proposal.Assignments
			taskDelegationMutex.Unlock()

			// Enhanced logging of delegation proposal details
			log.Printf("\n📋 DELEGATION PROPOSAL (Round 1) from %s:", v.Name)
			log.Printf("  Assignments proposed (%d):", len(proposal.Assignments))
			for subtask, assignedTo := range proposal.Assignments {
				log.Printf("  • \"%s\" → %s", subtask, assignedTo)
			}
			log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
			log.Printf("  -----------------------------")

			// Broadcast for UI
			communication.BroadcastEvent(communication.EventTaskDelegation, map[string]interface{}{
				"validatorId":   proposal.ValidatorID,
				"validatorName": proposal.ValidatorName,
				"assignments":   proposal.Assignments,
				"reasoning":     proposal.Reasoning,
				"round":         1,
				"blockHeight":   results.BlockInfo.Height,
				"timestamp":     time.Now(),
			})

			log.Printf("Validator %s submitted initial delegation proposal with %d assignments",
				v.Name, len(proposal.Assignments))
		}(validator)
	}

	round1Wg.Wait()
	results.DiscussionHistory[0] = TaskDelegationRound{
		Round:     1,
		Proposals: round1Proposals,
	}
	log.Printf("Completed Round 1 with %d delegation proposals", len(round1Proposals))

	// Wait between rounds
	time.Sleep(RoundDuration)

	// ROUND 2: Review and Critique Delegations
	// Agents review other delegation proposals and provide feedback
	log.Printf("Starting Round 2: Delegation Feedback and Refinement")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskDelegationRoundStart, map[string]interface{}{
		"round":       2,
		"blockHeight": results.BlockInfo.Height,
		"timestamp":   time.Now(),
	})

	round2Proposals := make(map[string]TaskDelegationProposal)
	var round2Wg sync.WaitGroup

	// Format round 1 proposals for context
	round1Context := formatDelegationProposals(round1Proposals, validators)

	for _, validator := range validators {
		round2Wg.Add(1)
		go func(v *Validator) {
			defer round2Wg.Done()

			proposal := generateDelegationFeedback(v, round1Context, results, validators)

			taskDelegationMutex.Lock()
			round2Proposals[v.ID] = proposal
			results.ValidatorVotes[v.ID] = proposal.Assignments
			taskDelegationMutex.Unlock()

			// Enhanced logging of delegation feedback details
			log.Printf("\n🔍 DELEGATION FEEDBACK (Round 2) from %s:", v.Name)
			log.Printf("  Refined assignments (%d):", len(proposal.Assignments))
			for subtask, assignedTo := range proposal.Assignments {
				log.Printf("  • \"%s\" → %s", subtask, assignedTo)
			}
			log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
			log.Printf("  -----------------------------")

			// Broadcast for UI
			communication.BroadcastEvent(communication.EventTaskDelegation, map[string]interface{}{
				"validatorId":   proposal.ValidatorID,
				"validatorName": proposal.ValidatorName,
				"assignments":   proposal.Assignments,
				"reasoning":     proposal.Reasoning,
				"round":         2,
				"blockHeight":   results.BlockInfo.Height,
				"timestamp":     time.Now(),
			})

			log.Printf("Validator %s submitted delegation feedback with %d assignments",
				v.Name, len(proposal.Assignments))
		}(validator)
	}

	round2Wg.Wait()
	results.DiscussionHistory[1] = TaskDelegationRound{
		Round:     2,
		Proposals: round2Proposals,
	}
	log.Printf("Completed Round 2 with %d delegation feedback proposals", len(round2Proposals))

	// Wait between rounds
	time.Sleep(RoundDuration)

	// ROUND 3: Final Delegation Decision
	// Agents continue discussions until they reach consensus
	log.Printf("Starting Round 3: Continuous Delegation Discussion Until Consensus")

	// Broadcast round start event
	communication.BroadcastEvent(communication.EventTaskDelegationRoundStart, map[string]interface{}{
		"round":       3,
		"blockHeight": results.BlockInfo.Height,
		"timestamp":   time.Now(),
	})

	// Define consensus parameters
	maxIterations := 5
	consensusThreshold := 0.75 // At least 75% consensus needed

	// Store all iterations of proposals
	var allRound3Proposals []map[string]TaskDelegationProposal
	var currentRound3Proposals map[string]TaskDelegationProposal
	var consensusReached bool
	var iteration int

	// Initial discussion context is from rounds 1 and 2
	discussionContext := formatDelegationHistory(results, validators)

	// Loop until consensus reached or max iterations
	for iteration = 0; iteration < maxIterations && !consensusReached; iteration++ {
		log.Printf("Starting delegation discussion iteration %d", iteration+1)

		currentRound3Proposals = make(map[string]TaskDelegationProposal)
		var iterationWg sync.WaitGroup

		// Current iteration context includes all previous round 3 discussions
		currentContext := discussionContext
		if iteration > 0 {
			// Add previous round 3 discussions to context
			currentContext += "\n\nPREVIOUS DISCUSSION ATTEMPTS:\n\n"
			for i, prevRoundProposals := range allRound3Proposals {
				currentContext += fmt.Sprintf("ITERATION %d:\n", i+1)
				currentContext += formatDelegationProposals(prevRoundProposals, validators)
				currentContext += "\n"
			}
		}

		// Each validator submits a proposal
		for _, validator := range validators {
			iterationWg.Add(1)
			go func(v *Validator) {
				defer iterationWg.Done()

				var proposal TaskDelegationProposal
				if iteration == 0 {
					// First iteration uses standard final decision function
					proposal = generateFinalDelegation(v, currentContext, results, validators)
				} else {
					// Subsequent iterations use consensus-building function
					proposal = generateDelegationConsensus(v, currentContext, results, validators, iteration)
				}

				taskDelegationMutex.Lock()
				currentRound3Proposals[v.ID] = proposal
				results.ValidatorVotes[v.ID] = proposal.Assignments
				taskDelegationMutex.Unlock()

				// Enhanced logging of delegation consensus details
				log.Printf("\n🔄 DELEGATION CONSENSUS (Round 3, Iteration %d) from %s:", iteration+1, v.Name)
				log.Printf("  Proposed assignments (%d):", len(proposal.Assignments))
				for subtask, assignedTo := range proposal.Assignments {
					log.Printf("  • \"%s\" → %s", subtask, assignedTo)
				}
				log.Printf("  Reasoning excerpt: %s", truncateString(proposal.Reasoning, 200))
				log.Printf("  -----------------------------")

				// Broadcast for UI
				communication.BroadcastEvent(communication.EventTaskDelegation, map[string]interface{}{
					"validatorId":   proposal.ValidatorID,
					"validatorName": proposal.ValidatorName,
					"assignments":   proposal.Assignments,
					"reasoning":     proposal.Reasoning,
					"round":         3,
					"iteration":     iteration + 1,
					"blockHeight":   results.BlockInfo.Height,
					"timestamp":     time.Now(),
				})

				log.Printf("Validator %s submitted delegation consensus proposal %d",
					v.Name, iteration+1)
			}(validator)
		}

		iterationWg.Wait()
		allRound3Proposals = append(allRound3Proposals, currentRound3Proposals)

		// Check for consensus
		finalAssignments := consolidateFinalDelegations(currentRound3Proposals, validators)
		consensusScore := calculateDelegationConsensusScore(currentRound3Proposals, finalAssignments)

		log.Printf("Delegation consensus iteration %d complete - consensus score: %.2f (threshold: %.2f)",
			iteration+1, consensusScore, consensusThreshold)

		// Broadcast iteration result
		communication.BroadcastEvent(communication.EventTaskDelegationRoundIteration, map[string]interface{}{
			"round":            3,
			"iteration":        iteration + 1,
			"consensusScore":   consensusScore,
			"threshold":        consensusThreshold,
			"consensusReached": consensusScore >= consensusThreshold,
			"blockHeight":      results.BlockInfo.Height,
			"timestamp":        time.Now(),
		})

		if consensusScore >= consensusThreshold {
			consensusReached = true
			log.Printf("Delegation consensus reached after %d iterations!", iteration+1)

			// Log detailed final delegation consensus
			log.Printf("\n====== FINAL TASK DELEGATION CONSENSUS DETAILS ======")
			log.Printf("Consensus Score: %.2f (Threshold: %.2f)", consensusScore, consensusThreshold)
			log.Printf("Iterations Required: %d of %d maximum", iteration+1, maxIterations)
			log.Printf("\nFinal agreed assignments (%d):", len(finalAssignments))
			for subtask, validator := range finalAssignments {
				log.Printf("• \"%s\" → %s", subtask, validator)
			}

			log.Printf("\nValidator Contributions:")
			for _, proposal := range currentRound3Proposals {
				numMatches := 0
				for subtask, consensusAssignee := range finalAssignments {
					if proposedAssignee, exists := proposal.Assignments[subtask]; exists &&
						proposedAssignee == consensusAssignee {
						numMatches++
					}
				}

				// Calculate match percentage
				matchPercentage := 0.0
				if len(finalAssignments) > 0 {
					matchPercentage = float64(numMatches) / float64(len(finalAssignments)) * 100
				}

				log.Printf("\n🧠 %s's contribution:", proposal.ValidatorName)
				log.Printf("  Consensus: %.1f%% (%d of %d assignments)",
					matchPercentage, numMatches, len(finalAssignments))
				log.Printf("  Full reasoning:")
				log.Printf("  %s", proposal.Reasoning)
			}

			// Move this section inside the consensus log
			// Initialize assignment frequency map for consensus history
			assignmentFrequency := make(map[string]map[string]int) // subtask -> (validator -> count)
			for _, subtask := range results.Subtasks {
				assignmentFrequency[subtask] = make(map[string]int)
			}

			// Count assignments from all rounds
			for _, proposal := range currentRound3Proposals {
				for subtask, validator := range proposal.Assignments {
					if _, exists := assignmentFrequency[subtask]; exists {
						assignmentFrequency[subtask][validator]++
					}
				}
			}

			log.Printf("\nFinal assignments with consensus history:")
			for subtask, assignedTo := range finalAssignments {
				// Get assignment counts for this subtask
				counts := assignmentFrequency[subtask]

				// Calculate total mentions
				totalMentions := 0
				for _, count := range counts {
					totalMentions += count
				}

				// Calculate consensus percentage
				consensusPct := 0.0
				if totalMentions > 0 {
					consensusPct = float64(counts[assignedTo]) / float64(totalMentions) * 100
				}

				log.Printf("Subtask: %s → Assignee: %s (Consensus: %.1f%%)",
					subtask, assignedTo, consensusPct)
			}

			// Log workload distribution in this section
			validatorWorkload := make(map[string]int)
			for _, validator := range finalAssignments {
				validatorWorkload[validator]++
			}

			log.Printf("\nWorkload Distribution:")
			for validator, count := range validatorWorkload {
				percentage := float64(count) / float64(len(finalAssignments)) * 100
				log.Printf("• %s: %d tasks (%.1f%%)", validator, count, percentage)
			}

			log.Printf("\n================================================")
		} else {
			// Wait between iterations
			time.Sleep(RoundDuration / 2)
		}
	}

	// Store the final round results
	results.DiscussionHistory[2] = TaskDelegationRound{
		Round:     3,
		Proposals: currentRound3Proposals,
	}

	// Consolidate the final assignments based on the final round
	finalAssignments := consolidateFinalDelegations(currentRound3Proposals, validators)

	if !consensusReached {
		log.Printf("WARNING: Max iterations (%d) reached without sufficient consensus. Using best available assignments.", maxIterations)
	}

	// If there are any unassigned tasks, assign them round-robin
	if len(finalAssignments) < len(results.Subtasks) {
		log.Printf("Some tasks were not assigned, assigning remaining tasks round-robin")
		assignRemainingTasks(finalAssignments, results.Subtasks, validators)
	}

	results.Assignments = finalAssignments

	// Add comprehensive summary information
	log.Printf("\n======= TASK DELEGATION SUMMARY =======")
	log.Printf("Process completed at: %s", time.Now().Format(time.RFC3339))
	log.Printf("Block Height: %d, Hash: %s", results.BlockInfo.Height, results.BlockInfo.Hash())
	log.Printf("Sufficient consensus achieved: %v (Score: %.2f)", consensusReached, calculateDelegationConsensusScore(currentRound3Proposals, finalAssignments))
	log.Printf("Rounds completed: %d standard + %d consensus iterations", 2, iteration)
	log.Printf("Validators participating: %d", len(validators))
	log.Printf("Subtasks delegated: %d", len(finalAssignments))

	// Log delegation statistics
	var totalProposals int
	assignmentFrequency := make(map[string]map[string]int) // subtask -> (validator -> count)

	// Initialize assignment frequency map
	for _, subtask := range results.Subtasks {
		assignmentFrequency[subtask] = make(map[string]int)
	}

	// Round 1
	for _, proposal := range results.DiscussionHistory[0].Proposals {
		totalProposals++
		for subtask, validator := range proposal.Assignments {
			if _, exists := assignmentFrequency[subtask]; exists {
				assignmentFrequency[subtask][validator]++
			}
		}
	}

	// Round 2
	for _, proposal := range results.DiscussionHistory[1].Proposals {
		totalProposals++
		for subtask, validator := range proposal.Assignments {
			if _, exists := assignmentFrequency[subtask]; exists {
				assignmentFrequency[subtask][validator]++
			}
		}
	}

	// Round 3 (all iterations)
	for _, iterProposals := range allRound3Proposals {
		for _, proposal := range iterProposals {
			totalProposals++
			for subtask, validator := range proposal.Assignments {
				if _, exists := assignmentFrequency[subtask]; exists {
					assignmentFrequency[subtask][validator]++
				}
			}
		}
	}

	// Calculate workload distribution
	validatorWorkload := make(map[string]int)
	for _, validator := range finalAssignments {
		validatorWorkload[validator]++
	}

	log.Printf("Total proposals generated: %d", totalProposals)

	log.Printf("\nWorkload Distribution:")
	for validator, count := range validatorWorkload {
		percentage := float64(count) / float64(len(finalAssignments)) * 100
		log.Printf("• %s: %d tasks (%.1f%%)", validator, count, percentage)
	}

	log.Printf("\nFinal assignments with consensus history:")
	for subtask, assignedTo := range finalAssignments {
		// Get assignment counts for this subtask
		counts := assignmentFrequency[subtask]

		// Calculate total mentions
		totalMentions := 0
		for _, count := range counts {
			totalMentions += count
		}

		// Calculate consensus percentage
		consensusPct := 0.0
		if totalMentions > 0 {
			consensusPct = float64(counts[assignedTo]) / float64(totalMentions) * 100
		}

		log.Printf("Subtask: %s → Assignee: %s (Consensus: %.1f%%)",
			subtask, assignedTo, consensusPct)
	}

	log.Printf("=======================================")

	// Broadcast final delegations
	communication.BroadcastEvent(communication.EventTaskDelegationFinal, map[string]interface{}{
		"assignments":      results.Assignments,
		"blockHeight":      results.BlockInfo.Height,
		"consensusReached": consensusReached,
		"iterationsNeeded": iteration,
		"timestamp":        time.Now(),
	})

	return results
}

// generateInitialDelegation creates an initial task delegation proposal from a validator
func generateInitialDelegation(v *Validator, results *TaskDelegationResults, validators []*Validator) TaskDelegationProposal {
	// Create a map of validator names for easy reference
	validatorNames := make([]string, len(validators))
	for i, validator := range validators {
		validatorNames[i] = validator.Name
	}

	validatorTraits := make(map[string][]string)
	for _, validator := range validators {
		validatorTraits[validator.Name] = validator.Traits
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 1 (Initial Delegation) of a collaborative task delegation process.

The following subtasks need to be delegated:
%s

Available validators and their traits:
%s

Your task is to PROPOSE ASSIGNMENTS for each subtask to the most suitable validator (including yourself).
Base your decision on each validator's traits and skills, and ensure a fair distribution of work.

Please respond with a JSON object containing:
{
  "assignments": {
    "Subtask 1": "Validator Name",
    "Subtask 2": "Validator Name",
    ...
  },
  "reasoning": "Your explanation of why you chose these assignments and your approach to task delegation"
}

Match validators to tasks where their strengths would be most valuable and distribute the workload fairly.`,
		v.Name, v.Traits, formatSubtasksList(results.Subtasks), formatValidatorsList(validators))

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var delegationData struct {
		Assignments map[string]string `json:"assignments"`
		Reasoning   string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &delegationData); err != nil {
		log.Printf("Error parsing initial delegation proposal from %s: %v", v.Name, err)
		// Fall back to a simple round-robin assignment if parsing fails
		delegationData.Assignments = make(map[string]string)
		for i, subtask := range results.Subtasks {
			validatorIndex := i % len(validators)
			delegationData.Assignments[subtask] = validators[validatorIndex].Name
		}
		delegationData.Reasoning = "Error parsing AI response, using round-robin assignment"
	}

	// Validate the assignments to ensure they reference existing validators
	validateAssignments(delegationData.Assignments, validatorNames)

	return TaskDelegationProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Assignments:   delegationData.Assignments,
		Reasoning:     delegationData.Reasoning,
		Timestamp:     time.Now(),
	}
}

// generateDelegationFeedback creates a proposal with feedback on other delegation proposals
func generateDelegationFeedback(v *Validator, proposalsContext string, results *TaskDelegationResults, validators []*Validator) TaskDelegationProposal {
	// Create a map of validator names for easy reference
	validatorNames := make([]string, len(validators))
	for i, validator := range validators {
		validatorNames[i] = validator.Name
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 2 (Feedback) of a collaborative task delegation process.

Subtasks to be delegated:
%s

Available validators and their traits:
%s

INITIAL DELEGATION PROPOSALS from validators:
%s

Your task is to REVIEW the initial delegation proposals from other validators, then:
1. CRITIQUE any assignments you think could be improved
2. SUPPORT assignments you think are strong matches
3. REFINE the assignments based on your expertise and the traits of the validators

Please respond with a JSON object containing:
{
  "feedback": "Your critique and/or support for other proposals",
  "assignments": {
    "Subtask 1": "Validator Name",
    "Subtask 2": "Validator Name",
    ...
  },
  "reasoning": "Explanation of your refined assignments and how they improve upon the initial proposals"
}

Consider workload balance, expertise matching, and efficiency in your feedback and refinements.`,
		v.Name, v.Traits, formatSubtasksList(results.Subtasks),
		formatValidatorsList(validators), proposalsContext)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var feedbackData struct {
		Feedback    string            `json:"feedback"`
		Assignments map[string]string `json:"assignments"`
		Reasoning   string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &feedbackData); err != nil {
		log.Printf("Error parsing delegation feedback from %s: %v", v.Name, err)
		// Fall back to copying the assignments from the first round
		feedbackData.Feedback = "Error parsing AI response"
		feedbackData.Assignments = make(map[string]string)

		// Get this validator's initial proposal if available
		if initialProposal, ok := results.DiscussionHistory[0].Proposals[v.ID]; ok {
			feedbackData.Assignments = initialProposal.Assignments
		} else {
			// Fall back to round-robin assignment
			for i, subtask := range results.Subtasks {
				validatorIndex := i % len(validators)
				feedbackData.Assignments[subtask] = validators[validatorIndex].Name
			}
		}

		feedbackData.Reasoning = "Error parsing AI response, using previous assignments"
	}

	// Validate the assignments to ensure they reference existing validators
	validateAssignments(feedbackData.Assignments, validatorNames)

	// Combine feedback and reasoning
	combinedReasoning := fmt.Sprintf("Feedback on proposals:\n%s\n\nReasoning for refinements:\n%s",
		feedbackData.Feedback, feedbackData.Reasoning)

	return TaskDelegationProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Assignments:   feedbackData.Assignments,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// generateFinalDelegation creates a final decision proposal based on all previous delegation discussion
func generateFinalDelegation(v *Validator, discussionContext string, results *TaskDelegationResults, validators []*Validator) TaskDelegationProposal {
	// Create a map of validator names for easy reference
	validatorNames := make([]string, len(validators))
	for i, validator := range validators {
		validatorNames[i] = validator.Name
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in Round 3 (Final Decision) of a collaborative task delegation process.

Subtasks to be delegated:
%s

Available validators and their traits:
%s

DISCUSSION HISTORY (Initial Proposals and Feedback):
%s

Your task is to make a FINAL DECISION on task delegation.
Use a consensus-building approach that aims to incorporate the most valuable aspects of all proposals.
Focus on identifying commonly proposed assignments across different validators' proposals.

When creating your final delegation assignments, prioritize:
- Assignments that appeared in multiple proposals (indicating broader consensus)
- Matching validators to tasks where there is strongest consensus on their fit
- Balancing the workload fairly across validators
- Maintaining logical groupings of related subtasks to the same validator

Please respond with a JSON object containing:
{
  "consensusStrategy": "Detailed description of how you're finding consensus among the delegation proposals",
  "assignments": {
    "Subtask 1": "Validator Name",
    "Subtask 2": "Validator Name",
    ...
  },
  "reasoning": "Explanation of why this delegation represents a good consensus"
}

Your assignments should represent the best consensus that can be achieved based on the discussion so far.`,
		v.Name, v.Traits, formatSubtasksList(results.Subtasks),
		formatValidatorsList(validators), discussionContext)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var decisionData struct {
		ConsensusStrategy string            `json:"consensusStrategy"`
		Assignments       map[string]string `json:"assignments"`
		Reasoning         string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &decisionData); err != nil {
		log.Printf("Error parsing final delegation decision from %s: %v", v.Name, err)
		// Fall back to copying the assignments from the second round
		decisionData.ConsensusStrategy = "Error parsing response"
		decisionData.Assignments = make(map[string]string)

		// Get this validator's feedback proposal if available
		if feedbackProposal, ok := results.DiscussionHistory[1].Proposals[v.ID]; ok {
			decisionData.Assignments = feedbackProposal.Assignments
		} else if initialProposal, ok := results.DiscussionHistory[0].Proposals[v.ID]; ok {
			// Fall back to initial proposal
			decisionData.Assignments = initialProposal.Assignments
		} else {
			// Fall back to round-robin assignment
			for i, subtask := range results.Subtasks {
				validatorIndex := i % len(validators)
				decisionData.Assignments[subtask] = validators[validatorIndex].Name
			}
		}

		decisionData.Reasoning = "Error parsing AI response, using previous assignments"
	}

	// Validate the assignments to ensure they reference existing validators
	validateAssignments(decisionData.Assignments, validatorNames)

	// Combine strategy and reasoning
	combinedReasoning := fmt.Sprintf("Consensus Strategy: %s\n\nReasoning:\n%s",
		decisionData.ConsensusStrategy, decisionData.Reasoning)

	return TaskDelegationProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Assignments:   decisionData.Assignments,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// formatDelegationProposals formats delegation proposals for review by other validators
func formatDelegationProposals(proposals map[string]TaskDelegationProposal, validators []*Validator) string {
	var result strings.Builder

	for _, proposal := range proposals {
		result.WriteString(fmt.Sprintf("Validator: %s\n", proposal.ValidatorName))
		result.WriteString("Assignments:\n")

		for subtask, assignedTo := range proposal.Assignments {
			result.WriteString(fmt.Sprintf("- \"%s\" → %s\n", subtask, assignedTo))
		}

		result.WriteString(fmt.Sprintf("Reasoning: %s\n\n", proposal.Reasoning))
	}

	return result.String()
}

// formatDelegationHistory formats the entire delegation discussion history for the final round
func formatDelegationHistory(results *TaskDelegationResults, validators []*Validator) string {
	var result strings.Builder

	// Round 1: Initial Proposals
	result.WriteString("ROUND 1 - INITIAL DELEGATION PROPOSALS:\n\n")
	result.WriteString(formatDelegationProposals(results.DiscussionHistory[0].Proposals, validators))

	// Round 2: Feedback
	result.WriteString("\nROUND 2 - DELEGATION FEEDBACK AND REFINEMENTS:\n\n")
	result.WriteString(formatDelegationProposals(results.DiscussionHistory[1].Proposals, validators))

	return result.String()
}

// formatSubtasksList creates a formatted list of subtasks
func formatSubtasksList(subtasks []string) string {
	var result strings.Builder

	for i, subtask := range subtasks {
		result.WriteString(fmt.Sprintf("%d. %s\n", i+1, subtask))
	}

	return result.String()
}

// formatValidatorsList creates a formatted list of validators with their traits
func formatValidatorsList(validators []*Validator) string {
	var result strings.Builder

	for _, validator := range validators {
		result.WriteString(fmt.Sprintf("- %s: %v\n", validator.Name, validator.Traits))
	}

	return result.String()
}

// validateAssignments ensures all assignments reference known validators
func validateAssignments(assignments map[string]string, validatorNames []string) {
	for subtask, assignedTo := range assignments {
		validAssignment := false
		for _, validName := range validatorNames {
			if assignedTo == validName {
				validAssignment = true
				break
			}
		}

		if !validAssignment {
			// If we find an invalid validator name, remove the assignment
			delete(assignments, subtask)
			log.Printf("WARNING: Removed invalid assignment to unknown validator: %s", assignedTo)
		}
	}
}

// consolidateFinalDelegations analyzes final delegation decisions and extracts the most agreed-upon assignments
func consolidateFinalDelegations(finalProposals map[string]TaskDelegationProposal, validators []*Validator) map[string]string {
	// For each subtask, count how many validators assigned it to each validator
	subtaskAssignmentCounts := make(map[string]map[string]int) // subtask -> (validatorName -> count)

	// Initialize the map for each subtask
	for _, proposal := range finalProposals {
		for subtask := range proposal.Assignments {
			if subtaskAssignmentCounts[subtask] == nil {
				subtaskAssignmentCounts[subtask] = make(map[string]int)
			}
		}
	}

	// Count assignments across all proposals
	for _, proposal := range finalProposals {
		for subtask, assignedTo := range proposal.Assignments {
			subtaskAssignmentCounts[subtask][assignedTo]++
		}
	}

	// For each subtask, find the validator with the most votes
	finalAssignments := make(map[string]string)

	for subtask, counts := range subtaskAssignmentCounts {
		var bestValidator string
		var maxCount int

		for validator, count := range counts {
			if count > maxCount {
				maxCount = count
				bestValidator = validator
			}
		}

		if bestValidator != "" {
			finalAssignments[subtask] = bestValidator
		}
	}

	log.Printf("Extracted %d final assignments from %d finalization proposals",
		len(finalAssignments), len(finalProposals))

	return finalAssignments
}

// assignRemainingTasks assigns any unassigned tasks using a round-robin approach
func assignRemainingTasks(assignments map[string]string, subtasks []string, validators []*Validator) {
	if len(validators) == 0 {
		return
	}

	// Count current assignments per validator to balance workload
	validatorTaskCount := make(map[string]int)
	for _, validator := range validators {
		validatorTaskCount[validator.Name] = 0
	}

	// Count existing assignments
	for _, assignedTo := range assignments {
		validatorTaskCount[assignedTo]++
	}

	// Find unassigned subtasks
	for _, subtask := range subtasks {
		if _, ok := assignments[subtask]; !ok {
			// Find the validator with the least tasks
			var leastBusyValidator string
			minTasks := -1

			for validator, count := range validatorTaskCount {
				if minTasks == -1 || count < minTasks {
					minTasks = count
					leastBusyValidator = validator
				}
			}

			// Assign the task to the least busy validator
			assignments[subtask] = leastBusyValidator
			validatorTaskCount[leastBusyValidator]++

			log.Printf("Assigned unassigned subtask '%s' to %s", subtask, leastBusyValidator)
		}
	}
}

// Helper min function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateDelegationConsensus creates a delegation proposal specifically aimed at building consensus
func generateDelegationConsensus(v *Validator, discussionContext string, results *TaskDelegationResults, validators []*Validator, iteration int) TaskDelegationProposal {
	// Create a map of validator names for easy reference
	validatorNames := make([]string, len(validators))
	for i, validator := range validators {
		validatorNames[i] = validator.Name
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in an EXTENDED Round 3 (Consensus Building) of a collaborative task delegation process.
This is iteration %d of the consensus-building process.

Subtasks to be delegated:
%s

Available validators and their traits:
%s

COMPLETE DISCUSSION HISTORY (including previous consensus attempts):
%s

Your task is to FIND CONSENSUS with the other validators on task delegation.
Review all previous proposals, especially the most recent iteration, and look for common ground.
Focus on refining and converging toward assignments that seem to have broader support.

Please respond with a JSON object containing:
{
  "consensusStrategy": "Explain how you're trying to bridge gaps between different delegation proposals",
  "assignments": {
    "Subtask 1": "Validator Name",
    "Subtask 2": "Validator Name",
    ...
  },
  "reasoning": "Explain why this delegation represents a good consensus and how it addresses the expertise needs"
}

Your goal is to help the group reach consensus on task assignments, not to push your own preferences.
Identify which assignments have broader support and adapt your proposal accordingly.`,
		v.Name, v.Traits, iteration+1, formatSubtasksList(results.Subtasks),
		formatValidatorsList(validators), discussionContext)

	// Log the delegation consensus-building prompt
	log.Printf("\n🔄 DELEGATION CONSENSUS PROMPT for %s (Iteration %d):\n%s\n", v.Name, iteration+1, prompt)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var consensusData struct {
		ConsensusStrategy string            `json:"consensusStrategy"`
		Assignments       map[string]string `json:"assignments"`
		Reasoning         string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &consensusData); err != nil {
		log.Printf("Error parsing delegation consensus proposal from %s: %v", v.Name, err)
		// Fall back to copying the assignments from a previous round
		consensusData.ConsensusStrategy = "Error parsing response"
		consensusData.Assignments = make(map[string]string)

		// Get this validator's previous proposal if available
		if iteration > 0 && len(results.DiscussionHistory) > 0 {
			previousIterationIndex := min(iteration, len(results.DiscussionHistory)-1)
			if previousProposal, ok := results.DiscussionHistory[previousIterationIndex].Proposals[v.ID]; ok {
				consensusData.Assignments = previousProposal.Assignments
			}
		}

		consensusData.Reasoning = "Error parsing AI response, using previous assignments"
	}

	// Validate the assignments to ensure they reference existing validators
	validateAssignments(consensusData.Assignments, validatorNames)

	// Combine strategy and reasoning
	combinedReasoning := fmt.Sprintf("Consensus Strategy (Iteration %d): %s\n\nReasoning:\n%s",
		iteration+1, consensusData.ConsensusStrategy, consensusData.Reasoning)

	return TaskDelegationProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Assignments:   consensusData.Assignments,
		Reasoning:     combinedReasoning,
		Timestamp:     time.Now(),
	}
}

// calculateDelegationConsensusScore measures how much consensus exists across delegation proposals
// Returns a value between 0 (no consensus) and 1 (perfect consensus)
func calculateDelegationConsensusScore(proposals map[string]TaskDelegationProposal, consensusAssignments map[string]string) float64 {
	if len(proposals) == 0 || len(consensusAssignments) == 0 {
		return 0.0
	}

	// For each validator, calculate what percentage of the consensus assignments they agreed with
	var totalConsensusScore float64

	for _, proposal := range proposals {
		// Count matching assignments
		var matches float64
		for subtask, consensusAssignee := range consensusAssignments {
			if proposedAssignee, exists := proposal.Assignments[subtask]; exists {
				if proposedAssignee == consensusAssignee {
					matches++
				}
			}
		}

		// Calculate consensus as percentage of consensus assignments matched
		consensusScore := matches / float64(len(consensusAssignments))
		totalConsensusScore += consensusScore
	}

	// Average consensus across all validators
	return totalConsensusScore / float64(len(proposals))
}

// NotifyAssignedValidators notifies validators of their assigned tasks
func NotifyAssignedValidators(chainID string, delegationResults *TaskDelegationResults) {
	if delegationResults == nil || len(delegationResults.Assignments) == 0 {
		log.Printf("No assignments to notify validators about")
		return
	}

	log.Printf("======= STARTING VALIDATOR TASK NOTIFICATIONS =======")
	log.Printf("Chain ID: %s", chainID)
	log.Printf("Block Height: %d", delegationResults.BlockInfo.Height)
	log.Printf("Block Hash: %s", delegationResults.BlockInfo.Hash())
	log.Printf("Total Assignments: %d", len(delegationResults.Assignments))
	log.Printf("---------------------------------------------------")

	// Get all validators for this chain
	validators := GetAllValidators(chainID)
	log.Printf("Found %d validators for this chain", len(validators))

	validatorMap := make(map[string]*Validator)
	for _, v := range validators {
		validatorMap[v.Name] = v
		log.Printf("Validator mapped: %s (ID: %s)", v.Name, v.ID)
	}

	// Group tasks by validator
	validatorTasks := make(map[string][]string)
	log.Printf("Assignment details:")
	for subtask, validatorName := range delegationResults.Assignments {
		validatorTasks[validatorName] = append(validatorTasks[validatorName], subtask)
		log.Printf("- Subtask: \"%s\" → Assigned to: %s", subtask, validatorName)
	}
	log.Printf("---------------------------------------------------")

	// Notify each validator of their assigned tasks
	log.Printf("Sending notifications to validators:")
	for validatorName, tasks := range validatorTasks {
		validator, exists := validatorMap[validatorName]
		if !exists {
			log.Printf("❌ ERROR: Cannot notify validator %s: not found in validator map", validatorName)
			continue
		}

		log.Printf("🔔 Notifying validator: %s (ID: %s)", validatorName, validator.ID)
		log.Printf("  Assigned tasks (%d):", len(tasks))
		for i, task := range tasks {
			log.Printf("  %d. %s", i+1, task)
		}

		// Create task notification payload
		taskNotification := map[string]interface{}{
			"validatorId":   validator.ID,
			"validatorName": validator.Name,
			"subtasks":      tasks,
			"blockHeight":   delegationResults.BlockInfo.Height,
			"blockHash":     delegationResults.BlockInfo.Hash(),
			"timestamp":     time.Now(),
		}

		// Broadcast task assignment event
		communication.BroadcastEvent(communication.EventTaskAssignment, taskNotification)
		log.Printf("  ✅ Assignment notification sent successfully via EventTaskAssignment")
	}

	log.Printf("======= VALIDATOR TASK NOTIFICATIONS COMPLETE =======")
}

// Helper function to truncate long strings for logging
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}
