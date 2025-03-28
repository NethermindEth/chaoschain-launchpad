package core

import (
	"fmt"
	"log"
)

// ProcessBlockTransactions processes all transactions within a block, including rewards
func ProcessBlockTransactions(block *Block) error {
	if block == nil {
		return fmt.Errorf("cannot process nil block")
	}

	log.Printf("Processing block %d with %d transactions", block.Height, len(block.Txs))

	// Get chain funds for this block's chain
	chainFunds := GetChainFunds(block.ChainID)
	if chainFunds == nil {
		// If not initialized, get the chain's reward pool
		chain := GetChain(block.ChainID)
		if chain == nil {
			return fmt.Errorf("chain %s not found", block.ChainID)
		}

		// Initialize with the chain's reward pool
		chainFunds = InitializeChainFunds(block.ChainID, float64(chain.RewardPool))
	}

	// Process all transactions in the block
	for i, tx := range block.Txs {
		log.Printf("Processing transaction %d of type %s", i+1, tx.Type)

		// Handle reward transactions differently
		if tx.Type == "REWARD" {
			if !ValidateRewardTransaction(&tx, block.ChainID) {
				log.Printf("Invalid reward transaction in block %d", block.Height)
				continue
			}

			// For a real implementation, recipients would be more sophisticated
			// For now, we'll assume the proposer gets the reward
			recipients := make(map[string]float64)
			recipients[block.Proposer] = tx.Reward

			// Process the reward
			if err := chainFunds.ProcessRewards(&tx, recipients); err != nil {
				log.Printf("Error processing reward transaction: %v", err)
				continue
			}

			log.Printf("Processed reward of %.2f to proposer %s", tx.Reward, block.Proposer)
		} else {
			// Process other transaction types
			log.Printf("Standard transaction processed: %s", tx.Type)
		}
	}

	return nil
}
