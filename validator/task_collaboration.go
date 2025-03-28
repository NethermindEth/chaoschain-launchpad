package validator

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

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

const (
	CollaborationRounds = 5               // Number of discussion rounds
	RoundDuration       = 5 * time.Second // Time per round
)

var (
	taskBreakdownMutex  sync.Mutex
	taskDelegationMutex sync.Mutex

	// Cache for semantic similarity results
	similarityCache      = make(map[string]bool)
	similarityCacheMutex sync.RWMutex

	// Batch processing for similarity comparisons
	batchQueue            = make(chan similarityRequest, 100)
	batchProcessorRunning atomic.Value
)

// similarityRequest represents a request to compare two subtasks
type similarityRequest struct {
	s1, s2     string
	resultChan chan bool
}

// Add an init function to properly initialize the atomic value
func init() {
	// Initialize atomic values
	batchProcessorRunning.Store(false)
}

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
		DiscussionHistory:  make([]TaskBreakdownRound, CollaborationRounds),
		ValidatorVotes:     make(map[string][]string),
		BlockInfo:          block,
		TransactionDetails: transactionDetails,
	}

	// Create a channel to collect validator proposals for each round
	proposalChannel := make(chan TaskBreakdownProposal)

	// Run multiple rounds of discussion
	for round := 1; round <= CollaborationRounds; round++ {
		log.Printf("Starting task breakdown round %d of %d", round, CollaborationRounds)
		results.DiscussionHistory[round-1] = TaskBreakdownRound{
			Round:     round,
			Proposals: make(map[string]TaskBreakdownProposal),
		}

		// Generate previous rounds context
		prevRoundsContext := ""
		if round > 1 {
			prevRoundsContext = getPreviousTaskBreakdownRoundsContext(results, round-1)
		}

		// Have each validator propose a breakdown
		var wg sync.WaitGroup
		for _, validator := range validators {
			wg.Add(1)
			go func(v *Validator) {
				defer wg.Done()
				proposal := generateTaskBreakdownProposal(v, round, prevRoundsContext, results)
				select {
				case proposalChannel <- proposal:
				case <-time.After(30 * time.Second):
					log.Printf("Timed out waiting to submit proposal for %s in round %d", v.Name, round)
				}
			}(validator)
		}

		// Close the proposal channel when all validators are done
		go func() {
			wg.Wait()
			close(proposalChannel)
		}()

		// Collect proposals from the channel
		for proposal := range proposalChannel {
			results.DiscussionHistory[round-1].Proposals[proposal.ValidatorID] = proposal

			// Also store in validator votes
			results.ValidatorVotes[proposal.ValidatorID] = proposal.Subtasks

			// Broadcast for UI
			communication.BroadcastEvent(communication.EventTaskBreakdown, map[string]interface{}{
				"validatorId":   proposal.ValidatorID,
				"validatorName": proposal.ValidatorName,
				"subtasks":      proposal.Subtasks,
				"reasoning":     proposal.Reasoning,
				"round":         round,
				"blockHeight":   block.Height,
				"timestamp":     time.Now(),
			})
		}

		log.Printf("Collected %d proposals for round %d",
			len(results.DiscussionHistory[round-1].Proposals), round)
	}

	// Consolidate into final set of subtasks
	finalSubtasks := consolidateTaskBreakdown(results)

	// If consolidation failed to produce any subtasks, extract some from the final round
	if len(finalSubtasks) == 0 {
		log.Printf("WARNING: Task breakdown consolidation produced zero subtasks. Using backup method.")

		// Try to extract at least some subtasks from the final round
		finalRound := results.DiscussionHistory[CollaborationRounds-1].Proposals
		if len(finalRound) > 0 {
			// Use the subtasks from the validator with the most comprehensive list
			maxSubtasks := 0
			var bestProposal TaskBreakdownProposal

			for _, proposal := range finalRound {
				if len(proposal.Subtasks) > maxSubtasks {
					maxSubtasks = len(proposal.Subtasks)
					bestProposal = proposal
				}
			}

			if maxSubtasks > 0 {
				log.Printf("Using subtasks from %s as a fallback", bestProposal.ValidatorName)
				finalSubtasks = bestProposal.Subtasks
			}
		}

		// If we still have no subtasks, create some generic ones
		if len(finalSubtasks) == 0 {
			log.Printf("Creating generic subtasks as a last resort")
			finalSubtasks = []string{
				"Research requirements and existing solutions",
				"Design system architecture",
				"Implement core functionality",
				"Test the implementation",
				"Deploy and document the solution",
			}
		}
	}

	results.FinalSubtasks = finalSubtasks
	log.Printf("Task breakdown completed with %d subtasks", len(finalSubtasks))

	if len(finalSubtasks) == 0 {
		log.Printf("Task breakdown process failed or produced no subtasks")
	}

	// Broadcast final breakdown
	communication.BroadcastEvent(communication.EventTaskBreakdownFinal, map[string]interface{}{
		"subtasks":    finalSubtasks,
		"blockHeight": block.Height,
		"timestamp":   time.Now(),
	})

	return results
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
		DiscussionHistory: make([]TaskDelegationRound, CollaborationRounds),
		ValidatorVotes:    make(map[string]map[string]string),
		BlockInfo:         taskBreakdown.BlockInfo,
		Subtasks:          taskBreakdown.FinalSubtasks,
	}

	// Run collaborative rounds
	for round := 1; round <= CollaborationRounds; round++ {
		log.Printf("Starting task delegation round %d/%d for block %d",
			round, CollaborationRounds, taskBreakdown.BlockInfo.Height)

		roundProposals := TaskDelegationRound{
			Round:     round,
			Proposals: make(map[string]TaskDelegationProposal),
		}

		// Gather previous rounds context
		previousRoundsContext := getPreviousTaskDelegationRoundsContext(results, round)

		// For each validator, generate a task delegation proposal
		var wg sync.WaitGroup
		for _, v := range validators {
			wg.Add(1)
			go func(v *Validator) {
				defer wg.Done()

				proposal := generateTaskDelegationProposal(v, round, previousRoundsContext, results, validators, taskBreakdown)

				// Add to round proposals
				taskDelegationMutex.Lock()
				roundProposals.Proposals[v.ID] = proposal
				taskDelegationMutex.Unlock()

				// Store in validator votes
				taskDelegationMutex.Lock()
				if results.ValidatorVotes[v.ID] == nil {
					results.ValidatorVotes[v.ID] = make(map[string]string)
				}
				for subtask, assignee := range proposal.Assignments {
					results.ValidatorVotes[v.ID][subtask] = assignee
				}
				taskDelegationMutex.Unlock()

				// Broadcast for UI
				communication.BroadcastEvent(communication.EventTaskDelegation, map[string]interface{}{
					"validatorId":   v.ID,
					"validatorName": v.Name,
					"assignments":   proposal.Assignments,
					"reasoning":     proposal.Reasoning,
					"round":         round,
					"blockHeight":   taskBreakdown.BlockInfo.Height,
					"timestamp":     time.Now(),
				})
			}(v)
		}

		// Wait for all validators to complete their proposals for this round
		wg.Wait()

		// Store this round in results
		results.DiscussionHistory[round-1] = roundProposals

		// Wait before starting next round
		time.Sleep(RoundDuration)
	}

	// After all rounds, consolidate the final task delegations
	results.Assignments = consolidateTaskDelegation(results)

	log.Printf("Task delegation completed with %d assignments", len(results.Assignments))

	// Broadcast final delegations
	communication.BroadcastEvent(communication.EventTaskDelegationFinal, map[string]interface{}{
		"assignments": results.Assignments,
		"blockHeight": taskBreakdown.BlockInfo.Height,
		"timestamp":   time.Now(),
	})

	return results
}

// NotifyAssignedValidators notifies validators of their assigned tasks
func NotifyAssignedValidators(chainID string, delegationResults *TaskDelegationResults) {
	validators := GetAllValidators(chainID)

	// Create a map of validator name to ID for lookup
	validatorNameToID := make(map[string]string)
	validatorIDToObject := make(map[string]*Validator)
	for _, v := range validators {
		validatorNameToID[v.Name] = v.ID
		validatorIDToObject[v.ID] = v
	}

	// Group tasks by validator
	validatorTasks := make(map[string][]string) // validator name -> list of tasks

	for subtask, validatorName := range delegationResults.Assignments {
		validatorTasks[validatorName] = append(validatorTasks[validatorName], subtask)
	}

	// Notify each validator of their assigned tasks
	for validatorName, tasks := range validatorTasks {
		validatorID := validatorNameToID[validatorName]
		if validatorID == "" {
			log.Printf("Unable to find validator ID for name: %s", validatorName)
			continue
		}

		v := validatorIDToObject[validatorID]
		if v == nil {
			log.Printf("Unable to find validator object for ID: %s", validatorID)
			continue
		}

		// Create task notification
		taskNotification := fmt.Sprintf("You have been assigned the following tasks for block %d:\n\n",
			delegationResults.BlockInfo.Height)

		for i, task := range tasks {
			taskNotification += fmt.Sprintf("%d. %s\n", i+1, task)
		}

		// Send notification to validator
		log.Printf("Notifying validator %s of %d assigned tasks", v.Name, len(tasks))

		// In a real system, this would send the notification to the validator
		// For now, we'll just log it
		log.Printf("Task notification for %s: %s", v.Name, taskNotification)

		// Broadcast assigned tasks event
		communication.BroadcastEvent(communication.EventTaskAssignment, map[string]interface{}{
			"validatorId":   v.ID,
			"validatorName": v.Name,
			"tasks":         tasks,
			"blockHeight":   delegationResults.BlockInfo.Height,
			"timestamp":     time.Now(),
		})
	}
}

// Helper functions

// generateTaskBreakdownProposal generates a task breakdown proposal from a validator
func generateTaskBreakdownProposal(v *Validator, round int, prevRoundsContext string,
	results *TaskBreakdownResults) TaskBreakdownProposal {

	// Create a variable for the "from previous rounds" text
	var prevRoundsText string
	if round > 1 {
		prevRoundsText = "from previous rounds"
	} else {
		prevRoundsText = ""
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in round %d of %d of a collaborative task breakdown process for the following:

%s

%s

Your task is to break down this request into clear, manageable subtasks. Consider:

1. What are the key components or modules needed?
2. What are the logical steps or phases required?
3. How should large tasks be divided into smaller, implementable pieces?
4. What dependencies exist between subtasks?

For round %d, please analyze both the original task and previous proposals %s.

If this is the first round, focus on comprehensive initial breakdown.
If this is a later round, focus on refining, combining, or improving previous proposals.
In the final round, aim for consensus on the most effective breakdown.

Please respond with a JSON object containing:
{
  "subtasks": ["Subtask 1 description", "Subtask 2 description", ...],
  "reasoning": "Your explanation of why you chose this breakdown and how it improves upon previous rounds"
}

Keep each subtask clear, specific, and implementable. Be concise but complete in your reasoning.`,
		v.Name, v.Traits, round, CollaborationRounds, results.TransactionDetails,
		getBlockContext(results.BlockInfo), round, prevRoundsText)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var proposalData struct {
		Subtasks  []string `json:"subtasks"`
		Reasoning string   `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &proposalData); err != nil {
		log.Printf("Error parsing task breakdown proposal from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		proposalData.Subtasks = []string{"Error parsing response"}
		proposalData.Reasoning = "Error parsing AI response"
	}

	// Log detailed information about the proposal
	log.Printf("======= TASK BREAKDOWN PROPOSAL (Round %d) =======", round)
	log.Printf("Validator: %s (%s)", v.Name, v.ID)
	log.Printf("Subtasks:")
	for i, subtask := range proposalData.Subtasks {
		log.Printf("  %d. %s", i+1, subtask)
	}
	log.Printf("Reasoning: %s", proposalData.Reasoning)
	log.Printf("=================================================")

	// Create and return the proposal
	return TaskBreakdownProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Subtasks:      proposalData.Subtasks,
		Reasoning:     proposalData.Reasoning,
		Timestamp:     time.Now(),
	}
}

// generateTaskDelegationProposal generates a task delegation proposal from a validator
func generateTaskDelegationProposal(v *Validator, round int, prevRoundsContext string,
	results *TaskDelegationResults, validators []*Validator,
	taskBreakdown *TaskBreakdownResults) TaskDelegationProposal {

	// Create a summary of available validators and their traits
	var validatorInfo strings.Builder
	for _, validator := range validators {
		validatorInfo.WriteString(fmt.Sprintf("- %s: Traits [%s], Expertise [%s]\n",
			validator.Name, strings.Join(validator.Traits, ", "), strings.Join(validator.Influences, ", ")))
	}

	// Create a list of subtasks
	var subtasksList strings.Builder
	for i, subtask := range results.Subtasks {
		subtasksList.WriteString(fmt.Sprintf("%d. %s\n", i+1, subtask))
	}

	var prevRoundsText string
	if round > 1 {
		prevRoundsText = "from previous rounds"
	} else {
		prevRoundsText = ""
	}

	prompt := fmt.Sprintf(`You are %s, with traits: %v.

You are participating in round %d of %d of a collaborative task delegation process.

The following subtasks need to be assigned to validators:
%s

Available validators and their traits:
%s

Original task context:
%s

%s

Your task is to assign each subtask to the most appropriate validator (including yourself) based on your judgement.

In this round %d, please analyze both the subtasks and previous delegation proposals %s.

If this is the first round, focus on making appropriate initial assignments.
If this is a later round, focus on refining assignments based on other validators' proposals.
In the final round, aim for consensus on the most effective assignments.

Please respond with a JSON object containing:
{
  "assignments": {"subtask1": "ValidatorName", "subtask2": "ValidatorName", ...},
  "reasoning": "Your explanation of why you chose these assignments and how they align with validator strengths"
}

Ensure every subtask is assigned, and assignments make logical sense based on skills and traits.`,
		v.Name, v.Traits, round, CollaborationRounds, subtasksList.String(),
		validatorInfo.String(), taskBreakdown.TransactionDetails,
		getBlockContext(results.BlockInfo), round, prevRoundsText)

	response := ai.GenerateLLMResponse(prompt)

	// Parse the response
	var proposalData struct {
		Assignments map[string]string `json:"assignments"`
		Reasoning   string            `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(response), &proposalData); err != nil {
		log.Printf("Error parsing task delegation proposal from %s: %v", v.Name, err)
		// Fall back to a simple structure if parsing fails
		proposalData.Assignments = make(map[string]string)
		proposalData.Reasoning = "Error parsing AI response"
	}

	// Log detailed information about the delegation proposal
	log.Printf("======= TASK DELEGATION PROPOSAL (Round %d) =======", round)
	log.Printf("Validator: %s (%s)", v.Name, v.ID)
	log.Printf("Assignments:")
	for subtask, assignee := range proposalData.Assignments {
		log.Printf("  '%s' → %s", subtask, assignee)
	}
	log.Printf("Reasoning: %s", proposalData.Reasoning)
	log.Printf("=================================================")

	// Create and return the proposal
	return TaskDelegationProposal{
		ValidatorID:   v.ID,
		ValidatorName: v.Name,
		Assignments:   proposalData.Assignments,
		Reasoning:     proposalData.Reasoning,
		Timestamp:     time.Now(),
	}
}

// getPreviousTaskBreakdownRoundsContext formats previous rounds' task breakdown proposals
func getPreviousTaskBreakdownRoundsContext(results *TaskBreakdownResults, currentRound int) string {
	if currentRound <= 1 {
		return "This is the first round, so there are no previous proposals."
	}

	var context strings.Builder
	context.WriteString("Previous rounds' proposals:\n\n")

	for round := 1; round < currentRound; round++ {
		if round-1 >= len(results.DiscussionHistory) {
			continue
		}

		roundData := results.DiscussionHistory[round-1]
		context.WriteString(fmt.Sprintf("Round %d:\n", round))

		for _, proposal := range roundData.Proposals {
			context.WriteString(fmt.Sprintf("Validator: %s\n", proposal.ValidatorName))
			context.WriteString("Subtasks:\n")

			for i, subtask := range proposal.Subtasks {
				context.WriteString(fmt.Sprintf("%d. %s\n", i+1, subtask))
			}

			context.WriteString(fmt.Sprintf("Reasoning: %s\n\n", proposal.Reasoning))
		}

		context.WriteString("\n")
	}

	return context.String()
}

// getPreviousTaskDelegationRoundsContext formats previous rounds' task delegation proposals
func getPreviousTaskDelegationRoundsContext(results *TaskDelegationResults, currentRound int) string {
	if currentRound <= 1 {
		return "This is the first round, so there are no previous proposals."
	}

	var context strings.Builder
	context.WriteString("Previous rounds' proposals:\n\n")

	for round := 1; round < currentRound; round++ {
		if round-1 >= len(results.DiscussionHistory) {
			continue
		}

		roundData := results.DiscussionHistory[round-1]
		context.WriteString(fmt.Sprintf("Round %d:\n", round))

		for _, proposal := range roundData.Proposals {
			context.WriteString(fmt.Sprintf("Validator: %s\n", proposal.ValidatorName))
			context.WriteString("Assignments:\n")

			for subtask, assignee := range proposal.Assignments {
				context.WriteString(fmt.Sprintf("- %s → %s\n", subtask, assignee))
			}

			context.WriteString(fmt.Sprintf("Reasoning: %s\n\n", proposal.Reasoning))
		}

		context.WriteString("\n")
	}

	return context.String()
}

// consolidateTaskBreakdown produces a final list of subtasks from all proposals
func consolidateTaskBreakdown(results *TaskBreakdownResults) []string {
	// Instead of exact matching, we'll group similar subtasks using semantic similarity
	type SubtaskInfo struct {
		Original     string   // Original text of subtask
		Normalized   string   // Normalized version for comparison
		Variants     []string // All variations of this subtask seen
		Count        int      // How many times something similar was proposed
		ValidatorIDs []string // Which validators proposed this or something similar
	}

	// List of subtask groups we've identified
	var subtaskGroups []SubtaskInfo

	// Process all proposals from all rounds, with focus on the final round
	// We'll weight the final round more heavily
	for roundIdx, round := range results.DiscussionHistory {
		// Weight increases for later rounds
		roundWeight := 1
		if roundIdx == CollaborationRounds-1 {
			// Final round has higher weight
			roundWeight = 3
		}

		for _, proposal := range round.Proposals {
			for _, subtask := range proposal.Subtasks {
				// Normalize for comparison
				normalizedSubtask := strings.TrimSpace(strings.ToLower(subtask))
				normalizedSubtask = removePrefixNumbers(normalizedSubtask)

				// Try to find an existing group this belongs to
				foundGroup := false
				for i := range subtaskGroups {
					// Check for similarity to existing groups
					if isSimilarSubtask(normalizedSubtask, subtaskGroups[i].Normalized) {
						subtaskGroups[i].Count += roundWeight
						subtaskGroups[i].Variants = append(subtaskGroups[i].Variants, subtask)

						// Only count each validator once per group
						if !contains(subtaskGroups[i].ValidatorIDs, proposal.ValidatorID) {
							subtaskGroups[i].ValidatorIDs = append(subtaskGroups[i].ValidatorIDs, proposal.ValidatorID)
						}

						foundGroup = true
						break
					}
				}

				// If no similar group found, create a new one
				if !foundGroup {
					subtaskGroups = append(subtaskGroups, SubtaskInfo{
						Original:     subtask,
						Normalized:   normalizedSubtask,
						Variants:     []string{subtask},
						Count:        roundWeight,
						ValidatorIDs: []string{proposal.ValidatorID},
					})
				}
			}
		}
	}

	// Sort groups by count (popularity)
	sort.Slice(subtaskGroups, func(i, j int) bool {
		// First by count
		if subtaskGroups[i].Count != subtaskGroups[j].Count {
			return subtaskGroups[i].Count > subtaskGroups[j].Count
		}
		// Then by number of validators who proposed it
		return len(subtaskGroups[i].ValidatorIDs) > len(subtaskGroups[j].ValidatorIDs)
	})

	// We'll take subtasks that were proposed by at least 2 validators
	// or were among the top 5-10 most frequent
	finalSubtasks := []string{}
	minValidators := 1

	// If we have enough validators, require agreement from at least 2
	if len(GetAllValidators(results.BlockInfo.ChainID)) >= 5 {
		minValidators = 2
	}

	log.Printf("======= TASK BREAKDOWN CONSOLIDATION ANALYSIS =======")
	log.Printf("Found %d distinct subtask groups", len(subtaskGroups))
	log.Printf("Minimum validators required: %d", minValidators)

	// First pass: include subtasks with broad agreement
	for i, group := range subtaskGroups {
		if len(group.ValidatorIDs) >= minValidators {
			// Choose the best representation from the variants
			bestVariant := chooseBestVariant(group.Variants)
			finalSubtasks = append(finalSubtasks, bestVariant)

			log.Printf("Group %d: '%s' - proposed by %d validators, count: %d - INCLUDED",
				i+1, bestVariant, len(group.ValidatorIDs), group.Count)
		} else {
			log.Printf("Group %d: '%s' - proposed by %d validators, count: %d - insufficient validator agreement",
				i+1, group.Original, len(group.ValidatorIDs), group.Count)
		}
	}

	// Second pass: if we don't have enough subtasks, include the most frequent ones
	if len(finalSubtasks) < 3 && len(subtaskGroups) > 0 {
		log.Printf("Not enough subtasks with broad agreement, including top frequent subtasks")

		// How many more subtasks do we need
		additionalNeeded := min(5, len(subtaskGroups)) - len(finalSubtasks)

		for i := 0; i < len(subtaskGroups) && additionalNeeded > 0; i++ {
			// Skip subtasks already included
			alreadyIncluded := false
			for _, existing := range finalSubtasks {
				if isSimilarSubtask(subtaskGroups[i].Normalized,
					strings.ToLower(removePrefixNumbers(existing))) {
					alreadyIncluded = true
					break
				}
			}

			if !alreadyIncluded {
				bestVariant := chooseBestVariant(subtaskGroups[i].Variants)
				finalSubtasks = append(finalSubtasks, bestVariant)
				additionalNeeded--

				log.Printf("Including additional subtask: '%s' due to frequency", bestVariant)
			}
		}
	}

	// If we still have no subtasks, create some generic ones
	if len(finalSubtasks) == 0 {
		log.Printf("Creating generic subtasks as a last resort")
		finalSubtasks = []string{
			"Research requirements and existing solutions",
			"Design system architecture",
			"Implement core functionality",
			"Test the implementation",
			"Deploy and document the solution",
		}
	}

	log.Printf("======= FINAL CONSOLIDATED TASK BREAKDOWN =======")
	log.Printf("Number of subtasks: %d", len(finalSubtasks))
	for i, subtask := range finalSubtasks {
		log.Printf("  %d. %s", i+1, subtask)
	}
	log.Printf("=================================================")

	return finalSubtasks
}

// Helper functions for semantic comparison

// isSimilarSubtask determines if two subtasks are semantically similar
func isSimilarSubtask(s1, s2 string) bool {
	// First use basic checks that don't require LLM calls to save on API costs

	// If strings are identical after normalization, they're similar
	if strings.TrimSpace(strings.ToLower(s1)) == strings.TrimSpace(strings.ToLower(s2)) {
		return true
	}

	// Check for containment (one is a subset of the other)
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		return true
	}

	// Compute Jaccard similarity between the word sets
	similarity := jaccardSimilarity(s1, s2)

	// If very similar by Jaccard, return true
	if similarity > 0.6 {
		return true
	}

	// For more complex comparisons, use the LLM
	// This is more expensive but more accurate for understanding semantics
	return llmSemanticSimilarity(s1, s2)
}

// llmSemanticSimilarity uses the LLM to determine if two subtasks are semantically similar
func llmSemanticSimilarity(s1, s2 string) bool {
	// Normalize inputs for consistency in caching
	normalizedS1 := strings.TrimSpace(strings.ToLower(s1))
	normalizedS2 := strings.TrimSpace(strings.ToLower(s2))

	// Ensure consistent order by sorting
	if normalizedS1 > normalizedS2 {
		normalizedS1, normalizedS2 = normalizedS2, normalizedS1
	}

	// Create a cache key
	cacheKey := normalizedS1 + "|||" + normalizedS2

	// Check cache first
	similarityCacheMutex.RLock()
	if result, exists := similarityCache[cacheKey]; exists {
		similarityCacheMutex.RUnlock()
		log.Printf("Cache hit for similarity comparison: '%s' vs '%s'",
			truncateString(s1, 30), truncateString(s2, 30))
		return result
	}
	similarityCacheMutex.RUnlock()

	// Not in cache, use batch processing
	return batchedSemanticSimilarity(s1, s2)
}

// truncateString ensures a string is no longer than the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// jaccardSimilarity computes the Jaccard similarity between two strings
// (size of intersection divided by size of union)
func jaccardSimilarity(s1, s2 string) float64 {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	// Create sets of words
	set1 := make(map[string]bool)
	for _, w := range words1 {
		set1[w] = true
	}

	set2 := make(map[string]bool)
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate intersection size
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	// Calculate union size
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// extractKeywords extracts important words from a string
func extractKeywords(s string) []string {
	// Split into words
	words := strings.Fields(s)

	// Filter out common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"to": true, "of": true, "for": true, "in": true, "on": true,
		"with": true, "by": true, "at": true, "from": true,
	}

	var keywords []string
	for _, word := range words {
		word = strings.ToLower(word)
		word = strings.Trim(word, ".,;:?!()\"-")

		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// chooseBestVariant selects the best representation from a list of variant subtasks
func chooseBestVariant(variants []string) string {
	if len(variants) == 0 {
		return ""
	}

	if len(variants) == 1 {
		return variants[0]
	}

	// Prefer longer, more descriptive variants
	bestVariant := variants[0]
	bestScore := len(strings.Fields(bestVariant))

	for _, variant := range variants[1:] {
		// Score based on length and presence of important keywords
		score := len(strings.Fields(variant))

		// Boost score for variants with more detailed descriptions
		if strings.Contains(strings.ToLower(variant), "implement") ||
			strings.Contains(strings.ToLower(variant), "design") ||
			strings.Contains(strings.ToLower(variant), "develop") ||
			strings.Contains(strings.ToLower(variant), "create") ||
			strings.Contains(strings.ToLower(variant), "build") {
			score += 1
		}

		if score > bestScore {
			bestScore = score
			bestVariant = variant
		}
	}

	return bestVariant
}

// contains checks if a string slice contains a specific string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// consolidateTaskDelegation produces final task assignments from all proposals
func consolidateTaskDelegation(results *TaskDelegationResults) map[string]string {
	// Data structure to track assignment votes
	type AssignmentVote struct {
		Subtask      string   // The subtask being assigned
		NormalizedST string   // Normalized version of subtask for comparison
		Validator    string   // The validator assigned to the task
		ValidatorIDs []string // Which validators voted for this assignment
		Count        int      // How many times this pairing was proposed
	}

	var assignmentVotes []AssignmentVote

	// Process all rounds of discussion, with higher weight for later rounds
	for roundIdx, round := range results.DiscussionHistory {
		// Weight increases for later rounds
		roundWeight := 1
		if roundIdx == CollaborationRounds-1 {
			// Final round has higher weight
			roundWeight = 3
		}

		for _, proposal := range round.Proposals {
			for subtask, validator := range proposal.Assignments {
				// Normalize the subtask for comparison
				normalizedSubtask := strings.TrimSpace(strings.ToLower(subtask))
				normalizedSubtask = removePrefixNumbers(normalizedSubtask)

				// Look for existing votes for this subtask-validator pair
				foundMatch := false
				for i := range assignmentVotes {
					// Check if this is a vote for the same assignment
					if isSimilarSubtask(normalizedSubtask, assignmentVotes[i].NormalizedST) &&
						assignmentVotes[i].Validator == validator {
						// Same assignment found, increment count
						assignmentVotes[i].Count += roundWeight

						// Only count each validator once per assignment
						if !contains(assignmentVotes[i].ValidatorIDs, proposal.ValidatorID) {
							assignmentVotes[i].ValidatorIDs = append(assignmentVotes[i].ValidatorIDs, proposal.ValidatorID)
						}

						foundMatch = true
						break
					}
				}

				// If no match found, create a new assignment vote
				if !foundMatch {
					assignmentVotes = append(assignmentVotes, AssignmentVote{
						Subtask:      subtask,
						NormalizedST: normalizedSubtask,
						Validator:    validator,
						ValidatorIDs: []string{proposal.ValidatorID},
						Count:        roundWeight,
					})
				}
			}
		}
	}

	// Group subtasks that are semantically similar
	type SubtaskGroup struct {
		OriginalSubtasks []string            // Different variations of this subtask
		NormalizedST     string              // A normalized form for comparison
		Assignments      map[string]int      // validator -> vote count
		ValidatorVoters  map[string][]string // validator -> validators who voted for it
	}

	// Create groups of similar subtasks
	var subtaskGroups []SubtaskGroup

	for _, vote := range assignmentVotes {
		// Try to find an existing group for this subtask
		foundGroup := false
		for i := range subtaskGroups {
			if isSimilarSubtask(vote.NormalizedST, subtaskGroups[i].NormalizedST) {
				// Add this vote to the existing group
				if _, exists := subtaskGroups[i].Assignments[vote.Validator]; !exists {
					subtaskGroups[i].Assignments[vote.Validator] = 0
					subtaskGroups[i].ValidatorVoters[vote.Validator] = []string{}
				}

				subtaskGroups[i].Assignments[vote.Validator] += vote.Count

				// Add unique validator IDs
				for _, validatorID := range vote.ValidatorIDs {
					if !contains(subtaskGroups[i].ValidatorVoters[vote.Validator], validatorID) {
						subtaskGroups[i].ValidatorVoters[vote.Validator] = append(
							subtaskGroups[i].ValidatorVoters[vote.Validator], validatorID)
					}
				}

				// Add this subtask text variant if it's not already in the group
				foundVariant := false
				for _, existingSubtask := range subtaskGroups[i].OriginalSubtasks {
					if existingSubtask == vote.Subtask {
						foundVariant = true
						break
					}
				}

				if !foundVariant {
					subtaskGroups[i].OriginalSubtasks = append(subtaskGroups[i].OriginalSubtasks, vote.Subtask)
				}

				foundGroup = true
				break
			}
		}

		// If no group found, create a new one
		if !foundGroup {
			newGroup := SubtaskGroup{
				OriginalSubtasks: []string{vote.Subtask},
				NormalizedST:     vote.NormalizedST,
				Assignments:      make(map[string]int),
				ValidatorVoters:  make(map[string][]string),
			}

			newGroup.Assignments[vote.Validator] = vote.Count
			newGroup.ValidatorVoters[vote.Validator] = vote.ValidatorIDs

			subtaskGroups = append(subtaskGroups, newGroup)
		}
	}

	// Log analysis of the groups
	log.Printf("======= TASK DELEGATION CONSOLIDATION ANALYSIS =======")
	log.Printf("Found %d subtask groups for delegation", len(subtaskGroups))

	// Final assignments map
	finalAssignments := make(map[string]string)

	// For each subtask group, find the validator with the most votes
	for i, group := range subtaskGroups {
		log.Printf("Group %d: Subtask variants: %v", i+1, group.OriginalSubtasks)

		// Find the best validator for this subtask
		var bestValidator string
		bestVotes := 0
		mostVoters := 0

		for validator, votes := range group.Assignments {
			voters := len(group.ValidatorVoters[validator])
			log.Printf("  Validator '%s': %d votes from %d validators", validator, votes, voters)

			// Choose based on number of votes first, then by number of unique validators
			if votes > bestVotes || (votes == bestVotes && voters > mostVoters) {
				bestValidator = validator
				bestVotes = votes
				mostVoters = voters
			}
		}

		// Only assign if we have a validator with at least two votes or a minimum count
		minVotes := 1
		if len(GetAllValidators(results.BlockInfo.ChainID)) >= 5 {
			minVotes = 2
		}

		if bestVotes >= minVotes {
			// Find the best representation of this subtask from the group
			bestSubtask := chooseBestVariant(group.OriginalSubtasks)

			// Try to match with an official subtask
			matchedSubtask := findBestMatchingSubtask(group.NormalizedST, results.Subtasks)
			if matchedSubtask != "" {
				finalAssignments[matchedSubtask] = bestValidator
				log.Printf("  ✓ ASSIGNED: '%s' → %s (matched to '%s')", bestSubtask, bestValidator, matchedSubtask)
			} else {
				finalAssignments[bestSubtask] = bestValidator
				log.Printf("  ✓ ASSIGNED: '%s' → %s", bestSubtask, bestValidator)
			}
		} else {
			log.Printf("  ✗ NOT ASSIGNED: Insufficient agreement (best: %d votes)", bestVotes)
		}
	}

	// Make sure all subtasks are assigned
	for _, subtask := range results.Subtasks {
		isAssigned := false
		for assignedSubtask := range finalAssignments {
			if isSimilarSubtask(
				strings.ToLower(removePrefixNumbers(subtask)),
				strings.ToLower(removePrefixNumbers(assignedSubtask))) {
				isAssigned = true
				break
			}
		}

		// If not assigned, assign it to the most common validator
		if !isAssigned {
			// Find the most commonly assigned validator
			validatorFreq := make(map[string]int)
			for _, validator := range finalAssignments {
				validatorFreq[validator]++
			}

			// Find the most common validator
			maxFreq := 0
			mostCommonValidator := ""
			for validator, freq := range validatorFreq {
				if freq > maxFreq {
					maxFreq = freq
					mostCommonValidator = validator
				}
			}

			// If we have at least one validator, assign this subtask to them
			if mostCommonValidator != "" {
				finalAssignments[subtask] = mostCommonValidator
				log.Printf("  ✓ AUTO-ASSIGNED: '%s' → %s (default assignment)", subtask, mostCommonValidator)
			}
		}
	}

	// Log the final consolidated task delegations
	log.Printf("======= FINAL CONSOLIDATED TASK DELEGATION =======")
	log.Printf("Number of assignments: %d", len(finalAssignments))
	for subtask, validator := range finalAssignments {
		log.Printf("  '%s' → %s", subtask, validator)
	}
	log.Printf("=================================================")

	return finalAssignments
}

// Helper function to find the best matching subtask from the original list
func findBestMatchingSubtask(normalizedSubtask string, originalSubtasks []string) string {
	// First try exact match after normalization
	for _, original := range originalSubtasks {
		normalizedOriginal := strings.TrimSpace(strings.ToLower(original))
		normalizedOriginal = removePrefixNumbers(normalizedOriginal)

		if normalizedOriginal == normalizedSubtask {
			return original
		}
	}

	// If no exact match, try to find the closest match
	// This handles cases where subtasks might have been reworded slightly
	bestMatch := ""
	highestSimilarity := 0.0

	for _, original := range originalSubtasks {
		normalizedOriginal := strings.TrimSpace(strings.ToLower(original))
		normalizedOriginal = removePrefixNumbers(normalizedOriginal)

		similarity := calculateSimilarity(normalizedOriginal, normalizedSubtask)
		if similarity > highestSimilarity {
			highestSimilarity = similarity
			bestMatch = original
		}
	}

	// Only return if the similarity is above a threshold
	if highestSimilarity > 0.7 {
		return bestMatch
	}

	return ""
}

// Helper function to calculate similarity between two strings
func calculateSimilarity(s1, s2 string) float64 {
	// Simple word overlap for now
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	// Count common words
	commonWords := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				commonWords++
				break
			}
		}
	}

	// Calculate similarity as proportion of common words
	totalWords := len(words1) + len(words2)
	if totalWords == 0 {
		return 0
	}

	return float64(2*commonWords) / float64(totalWords)
}

// Helper max function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getBlockContext returns formatted block information
func getBlockContext(block *core.Block) string {
	return fmt.Sprintf("Block Information:\n"+
		"- Height: %d\n"+
		"- Hash: %s\n"+
		"- Proposer: %s\n"+
		"- Timestamp: %d\n",
		block.Height, block.Hash(), block.Proposer, block.Timestamp)
}

// Helper function to remove prefix numbers from subtasks
func removePrefixNumbers(subtask string) string {
	// Remove patterns like "1. ", "1) ", "1- ", "Step 1: " etc.
	// First try with regex
	re := regexp.MustCompile(`^(\d+\.?\s*|\d+\)\s*|\d+-\s*|step\s+\d+:?\s*|task\s+\d+:?\s*)`)
	cleaned := re.ReplaceAllString(subtask, "")

	// If changed, return the cleaned version
	if cleaned != subtask {
		return cleaned
	}

	// Otherwise, try to find the first non-numeric, non-punctuation character
	for i, char := range subtask {
		if !unicode.IsDigit(char) && !unicode.IsPunct(char) && !unicode.IsSpace(char) {
			// Found the first real content character
			return subtask[i:]
		}
	}

	// If we couldn't find anything to clean, return as is
	return subtask
}

// startBatchProcessor starts the background processor for batched similarity requests
func startBatchProcessor() {
	// Initialize the atomic value if needed
	var isRunning bool
	if v := batchProcessorRunning.Load(); v != nil {
		isRunning = v.(bool)
	}

	// Only start if not already running
	if !isRunning {
		// Try to set the flag atomically
		batchProcessorRunning.Store(true)
		go processSimilarityBatches()
	}
}

// processSimilarityBatches processes similarity requests in batches
func processSimilarityBatches() {
	defer batchProcessorRunning.Store(false)

	batchSize := 5                         // Process up to 5 comparisons at once
	batchTimeout := 500 * time.Millisecond // Wait up to 500ms to collect a batch

	for {
		// Collect a batch of requests
		var batch []similarityRequest
		timeoutTimer := time.NewTimer(batchTimeout)

		// Get the first request or timeout
		select {
		case req := <-batchQueue:
			batch = append(batch, req)
			timeoutTimer.Stop()
		case <-timeoutTimer.C:
			// No requests, try again
			if len(batch) == 0 {
				// No activity, exit processor after timeout
				log.Printf("Batch similarity processor shutting down due to inactivity")
				return
			}
		}

		// Try to fill the batch with remaining capacity
		batchComplete := false
		for len(batch) < batchSize && !batchComplete {
			select {
			case req := <-batchQueue:
				batch = append(batch, req)
			default:
				batchComplete = true
			}
		}

		// If we got at least one request, process the batch
		if len(batch) > 0 {
			processSimilarityBatch(batch)
		}
	}
}

// processSimilarityBatch sends a batch of comparisons to the LLM
func processSimilarityBatch(batch []similarityRequest) {
	if len(batch) == 1 {
		// If only one request, handle it directly
		result := llmSemanticSimilaritySingle(batch[0].s1, batch[0].s2)
		batch[0].resultChan <- result
		return
	}

	// Build a prompt for multiple comparisons
	var promptBuilder strings.Builder
	promptBuilder.WriteString("For each pair of task descriptions, determine if they are semantically similar.\n\n")
	promptBuilder.WriteString("Two tasks are semantically similar if they:\n")
	promptBuilder.WriteString("1. Refer to the same core activity or objective\n")
	promptBuilder.WriteString("2. Could reasonably be consolidated into a single task\n")
	promptBuilder.WriteString("3. Would be redundant if both were included in a task list\n\n")

	promptBuilder.WriteString("Analyze each pair carefully and respond with ONLY the word 'SIMILAR' or 'DIFFERENT' for each numbered pair.\n\n")

	for i, req := range batch {
		promptBuilder.WriteString(fmt.Sprintf("%d. Pair:\n", i+1))
		promptBuilder.WriteString(fmt.Sprintf("Task A: %s\n", req.s1))
		promptBuilder.WriteString(fmt.Sprintf("Task B: %s\n\n", req.s2))
	}

	// Get response from LLM
	response := ai.GenerateLLMResponse(promptBuilder.String())

	// Parse the response
	lines := strings.Split(response, "\n")
	results := make(map[int]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Look for lines with pattern like "1. SIMILAR" or "1: DIFFERENT"
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '.' || r == ':' || r == '-' || r == ' '
		})

		if len(parts) >= 2 {
			// Try to extract the number
			if num, err := strconv.Atoi(parts[0]); err == nil && num >= 1 && num <= len(batch) {
				// Look for "similar" in the response
				responseLower := strings.ToLower(line)
				results[num] = strings.Contains(responseLower, "similar")
			}
		}
	}

	// Send results to channels and cache
	for i, req := range batch {
		resultNum := i + 1
		result, found := results[resultNum]

		if !found {
			// Default to using single comparison as fallback
			log.Printf("Couldn't parse batch result for pair %d, falling back to single comparison", resultNum)
			result = llmSemanticSimilaritySingle(req.s1, req.s2)
		} else {
			// Cache this result
			normalizedS1 := strings.TrimSpace(strings.ToLower(req.s1))
			normalizedS2 := strings.TrimSpace(strings.ToLower(req.s2))

			// Ensure consistent order
			if normalizedS1 > normalizedS2 {
				normalizedS1, normalizedS2 = normalizedS2, normalizedS1
			}

			cacheKey := normalizedS1 + "|||" + normalizedS2

			similarityCacheMutex.Lock()
			similarityCache[cacheKey] = result
			similarityCacheMutex.Unlock()

			log.Printf("Batch similarity comparison %d: '%s' vs '%s' → %v",
				resultNum, truncateString(req.s1, 30), truncateString(req.s2, 30), result)
		}

		req.resultChan <- result
	}
}

// batchedSemanticSimilarity queues a similarity request for batch processing
func batchedSemanticSimilarity(s1, s2 string) bool {
	// Make sure the processor is running
	startBatchProcessor()

	// Create a channel for the result
	resultChan := make(chan bool, 1)

	// Send the request to the batch queue
	batchQueue <- similarityRequest{
		s1:         s1,
		s2:         s2,
		resultChan: resultChan,
	}

	// Wait for the result
	return <-resultChan
}

// llmSemanticSimilaritySingle uses the LLM to process a single comparison
func llmSemanticSimilaritySingle(s1, s2 string) bool {
	prompt := fmt.Sprintf(`Compare the following two task descriptions and determine if they are semantically similar:

Task 1: %s
Task 2: %s

Two tasks are considered semantically similar if they:
1. Refer to the same core activity or objective
2. Could reasonably be consolidated into a single task
3. Would be redundant if both were included in a task list

Analyze the two tasks carefully. Consider:
- The core action or objective
- The subject matter or domain
- The implied skills or knowledge required
- The scope and expected output

Reply with ONLY "similar" or "different".`, s1, s2)

	// Get response from LLM
	response := ai.GenerateLLMResponse(prompt)

	// Log the comparison for debugging
	log.Printf("Single LLM Similarity comparison: '%s' vs '%s' → %s",
		truncateString(s1, 30), truncateString(s2, 30), response)

	// Check the response
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "similar" || strings.Contains(response, "similar")
}
