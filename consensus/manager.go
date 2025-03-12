package consensus

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/core"
)

type ConsensusState int

const (
	Pending ConsensusState = iota
	InDiscussion
	Finalizing
	Accepted
	Rejected
	DiscussionTimeout = 30 * time.Second // Time allowed for discussion
	MinimumValidators = 2                // Minimum validators needed for consensus
)

type BlockConsensus struct {
	Block       *core.Block
	State       ConsensusState
	Votes       map[string]bool // validator ID -> vote
	StartTime   time.Time
	Discussions []Discussion
	mu          sync.RWMutex
}

type ConsensusResult struct {
	State   ConsensusState
	Support int
	Oppose  int
}

type ConsensusManager struct {
	chainID         string
	activeConsensus *BlockConsensus
	subscribers     map[int64][]chan ConsensusResult // blockHeight -> channels
	mu              sync.RWMutex
}

var (
	managers     = make(map[string]*ConsensusManager)
	managersLock sync.RWMutex
)

// GetConsensusManager returns the singleton consensus manager
func GetConsensusManager(chainID string) *ConsensusManager {
	managersLock.Lock()
	defer managersLock.Unlock()

	if manager, exists := managers[chainID]; exists {
		return manager
	}

	manager := &ConsensusManager{
		chainID:     chainID,
		subscribers: make(map[int64][]chan ConsensusResult),
	}
	managers[chainID] = manager
	return manager
}

// ProposeBlock starts the consensus process for a new block
func (cm *ConsensusManager) ProposeBlock(block *core.Block) error {
	// Validate block belongs to this chain
	if block.ChainID != cm.chainID {
		return fmt.Errorf("invalid block: wrong chain ID")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if there's already an active consensus
	if cm.activeConsensus != nil && cm.activeConsensus.State != Accepted && cm.activeConsensus.State != Rejected {
		return fmt.Errorf("another consensus is already in progress")
	}

	// Create new consensus for the block
	cm.activeConsensus = &BlockConsensus{
		Block:       block,
		State:       Pending,
		Votes:       make(map[string]bool),
		StartTime:   time.Now(),
		Discussions: make([]Discussion, 0),
	}

	// Start consensus process
	go cm.runConsensusProcess()

	return nil
}

// runConsensusProcess manages the lifecycle of block consensus
func (cm *ConsensusManager) runConsensusProcess() {
	// Move to discussion phase
	cm.activeConsensus.mu.Lock()
	cm.activeConsensus.State = InDiscussion
	cm.activeConsensus.mu.Unlock()

	// Trigger discussion rounds
	blockData, err := json.Marshal(cm.activeConsensus.Block)
	if err != nil {
		log.Printf("Failed to marshal block: %v", err)
		return
	}

	// Publish discussion trigger
	err = core.NatsBrokerInstance.Publish("BLOCK_DISCUSSION_TRIGGER", blockData)
	if err != nil {
		log.Printf("Failed to publish discussion trigger: %v", err)
		return
	}

	// Wait for all discussion rounds plus voting round
	totalTime := time.Duration(DiscussionRounds+1) * RoundDuration
	time.Sleep(totalTime)

	// Add additional buffer time for last votes to arrive
	time.Sleep(5 * time.Second) // Buffer for vote collection

	// Move to finalization phase
	cm.activeConsensus.mu.Lock()
	cm.activeConsensus.State = Finalizing

	// Get final consensus state
	consensus := cm.GetActiveConsensus()
	if consensus == nil {
		return
	}

	bc := core.GetChain(consensus.Block.ChainID)
	if bc == nil {
		return
	}

	// Count votes
	support := 0
	oppose := 0
	for _, d := range consensus.Discussions {
		if d.Round == DiscussionRounds+1 { // Only count final votes
			if strings.ToLower(d.Type) == "support" {
				support++
			} else if strings.ToLower(d.Type) == "oppose" {
				oppose++
			}
		}
	}

	// Make final decision
	totalVotes := support + oppose
	if totalVotes < MinimumValidators {
		cm.activeConsensus.State = Rejected
		// Return transactions to mempool
		for _, tx := range cm.activeConsensus.Block.Txs {
			bc.Mempool.AddTransaction(tx)
		}
	} else if float64(support)/float64(totalVotes) > 0.5 {
		cm.activeConsensus.State = Accepted
		// Add block to blockchain
		if err := bc.AddBlock(*cm.activeConsensus.Block); err != nil {
			log.Printf("Failed to add accepted block: %v", err)
			cm.activeConsensus.State = Rejected
			// Return transactions to mempool on failure
			for _, tx := range cm.activeConsensus.Block.Txs {
				bc.Mempool.AddTransaction(tx)
			}
		} else {
			// Clear processed transactions from mempool
			bc.Mempool.CleanupExpiredTransactions()
		}
	} else {
		cm.activeConsensus.State = Rejected
		// Return transactions to mempool
		for _, tx := range cm.activeConsensus.Block.Txs {
			bc.Mempool.AddTransaction(tx)
		}
	}

	// Broadcast results
	result := ConsensusResult{
		State:   cm.activeConsensus.State,
		Support: support,
		Oppose:  oppose,
	}

	// Broadcast verdict
	communication.BroadcastEvent(communication.EventBlockVerdict, result)

	// Broadcast detailed voting result
	votingResult := struct {
		BlockHeight int64          `json:"blockHeight"`
		State       ConsensusState `json:"state"`
		Support     int            `json:"support"`
		Oppose      int            `json:"oppose"`
		Accepted    bool           `json:"accepted"`
		Reason      string         `json:"reason"`
	}{
		BlockHeight: int64(cm.activeConsensus.Block.Height),
		State:       cm.activeConsensus.State,
		Support:     support,
		Oppose:      oppose,
		Accepted:    cm.activeConsensus.State == Accepted,
		Reason:      getConsensusReason(support, oppose, totalVotes),
	}
	communication.BroadcastEvent(communication.EventVotingResult, votingResult)

	// Notify subscribers
	cm.notifySubscribers(int64(cm.activeConsensus.Block.Height), result)
	cm.activeConsensus.mu.Unlock()
}

func getConsensusReason(support, oppose, total int) string {
	if total < MinimumValidators {
		return "Insufficient validator participation"
	}
	if float64(support)/float64(total) > 0.5 {
		return "Majority support achieved"
	}
	return "Insufficient support"
}

// GetActiveConsensus returns the current consensus state
func (cm *ConsensusManager) GetActiveConsensus() *BlockConsensus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.activeConsensus
}

// SubscribeResult allows waiting for consensus completion
func (cm *ConsensusManager) SubscribeResult(blockHeight int64, ch chan ConsensusResult) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.subscribers == nil {
		cm.subscribers = make(map[int64][]chan ConsensusResult)
	}
	cm.subscribers[blockHeight] = append(cm.subscribers[blockHeight], ch)
}

// notifySubscribers sends result to all subscribers
func (cm *ConsensusManager) notifySubscribers(height int64, result ConsensusResult) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if subs, ok := cm.subscribers[height]; ok {
		for _, ch := range subs {
			ch <- result
			close(ch)
		}
		delete(cm.subscribers, height)
	}
}
