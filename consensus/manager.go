package consensus

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

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
	activeConsensus *BlockConsensus
	subscribers     map[int64][]chan ConsensusResult // blockHeight -> channels
	mu              sync.RWMutex
}

var (
	defaultManager *ConsensusManager
	managerOnce    sync.Once
)

// GetConsensusManager returns the singleton consensus manager
func GetConsensusManager() *ConsensusManager {
	managerOnce.Do(func() {
		defaultManager = &ConsensusManager{}
	})
	return defaultManager
}

// ProposeBlock starts the consensus process for a new block
func (cm *ConsensusManager) ProposeBlock(block *core.Block) error {
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

	// Marshal the block data to JSON.
	blockData, err := json.Marshal(cm.activeConsensus.Block)
	if err != nil {
		log.Printf("Failed to marshal block: %v", err)
	} else {
		// Publish a discussion trigger event through NATS.
		err = core.NatsBrokerInstance.Publish("BLOCK_DISCUSSION_TRIGGER", blockData)
		if err != nil {
			log.Printf("Failed to publish discussion trigger: %v", err)
		} else {
			log.Println("Published BLOCK_DISCUSSION_TRIGGER event")
		}
	}

	// Wait for discussion period to allow validators to process the trigger and debate the block
	time.Sleep(DiscussionTimeout)

	// Move to finalization phase
	cm.activeConsensus.mu.Lock()
	cm.activeConsensus.State = Finalizing
	log.Println("Finalization phase started for block", cm.activeConsensus.Block.Height)

	// Count support vs opposition
	var support, oppose int
	for _, discussion := range cm.activeConsensus.Discussions {
		if discussion.Type == "support" {
			support++
		} else if discussion.Type == "oppose" {
			oppose++
		}
	}
	log.Printf("Votes count: support=%d, oppose=%d", support, oppose)

	// Make final decision based on votes
	totalVotes := support + oppose
	if totalVotes < MinimumValidators {
		log.Println("Not enough votes; rejecting block")
		cm.activeConsensus.State = Rejected
		cm.activeConsensus.mu.Unlock()

		// Return transactions to mempool
		for _, tx := range cm.activeConsensus.Block.Txs {
			core.GetBlockchain().Mempool.AddTransaction(tx)
		}
		return
	}

	// Need more than 50% support to accept
	if float64(support)/float64(totalVotes) > 0.5 {
		cm.activeConsensus.State = Accepted
		log.Println("Consensus reached with sufficient support; adding block to blockchain")
		// Add block to blockchain only if consensus is reached
		if err := core.GetBlockchain().AddBlock(*cm.activeConsensus.Block); err != nil {
			log.Printf("Failed to add accepted block: %v", err)
			cm.activeConsensus.State = Rejected
			// Return transactions to mempool on failure
			for _, tx := range cm.activeConsensus.Block.Txs {
				core.GetBlockchain().Mempool.AddTransaction(tx)
			}
		} else {
			// Clear processed transactions from mempool only on successful addition
			cm.activeConsensus.Block.Txs = nil // Help GC
			core.GetBlockchain().Mempool.CleanupExpiredTransactions()
		}
	} else {
		log.Println("Insufficient support; rejecting block")
		cm.activeConsensus.State = Rejected
		// Return transactions to mempool
		for _, tx := range cm.activeConsensus.Block.Txs {
			core.GetBlockchain().Mempool.AddTransaction(tx)
		}
	}
	cm.activeConsensus.mu.Unlock()

	// Notify subscribers and broadcast result
	result := ConsensusResult{
		State:   cm.activeConsensus.State,
		Support: support,
		Oppose:  oppose,
	}
	log.Printf("Final consensus result for block %d: %+v", cm.activeConsensus.Block.Height, result)
	cm.notifySubscribers(int64(cm.activeConsensus.Block.Height), result)
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
