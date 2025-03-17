package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/consensus"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	da "github.com/NethermindEth/chaoschain-launchpad/da_layer"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/registry"
	"github.com/NethermindEth/chaoschain-launchpad/storage"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

var (
	lastUsedPort         = 8080
	portMutex            sync.Mutex
	agentIdentitiesMutex sync.RWMutex
)

func findAvailablePort() int {
	portMutex.Lock()
	defer portMutex.Unlock()
	lastUsedPort++
	return lastUsedPort
}

func findAvailableAPIPort() int {
	portMutex.Lock()
	defer portMutex.Unlock()
	lastUsedPort++
	return lastUsedPort
}

// Add at the top with other types
type RelationshipUpdate struct {
	FromID   string  `json:"fromId"`
	TargetID string  `json:"targetId"`
	Score    float64 `json:"score"` // -1.0 to 1.0
}

// RegisterAgent - Registers a new AI agent (Producer or Validator)
func RegisterAgent(c *gin.Context) {
	chainID := c.GetString("chainID")
	chain := core.GetChain(chainID)
	if chain == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	var agent core.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}

	// Assign a unique ID
	agent.ID = uuid.New().String()

	// Get bootstrap node's P2P instance
	var bootstrapNode *p2p.Node
	chain.NodesMu.RLock()
	for _, node := range chain.Nodes {
		bootstrapNode = node
		break // Get first node
	}
	chain.NodesMu.RUnlock()

	if bootstrapNode == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No bootstrap node found for chain"})
		return
	}

	bootstrapPort := bootstrapNode.GetPort()
	if bootstrapPort == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bootstrap node not ready"})
		return
	}

	log.Printf("Found bootstrap node at port: %d", bootstrapPort)

	// Create a new node for this agent
	newPort := findAvailablePort()
	agentNode := node.NewNode(node.NodeConfig{
		ChainConfig: p2p.ChainConfig{
			ChainID: chainID,
			P2PPort: newPort,
			APIPort: findAvailableAPIPort(),
		},
		BootstrapNode: fmt.Sprintf("localhost:%d", bootstrapPort),
	})

	if err := agentNode.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start agent node"})
		return
	}

	// Register the new node with the correct chain
	addr := fmt.Sprintf("localhost:%d", newPort)

	chain.RegisterNode(addr, agentNode.GetP2PNode())

	if agent.Role == "producer" {
		personality := ai.Personality{
			Name:   agent.Name,
			Traits: agent.Traits,
			Style:  agent.Style,
		}

		// Get mempool safely
		mp, ok := agentNode.GetMempool().(*mempool.Mempool)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid mempool type"})
			return
		}

		// Create producer on its own node
		producerInstance := producer.NewProducer(mp, personality, agentNode.GetP2PNode())

		// Register on the agent's node
		registry.RegisterProducer(chainID, agent.ID, producerInstance)

	} else if agent.Role == "validator" {
		validatorInstance := validator.NewValidator(
			agent.ID,
			agent.Name,
			agent.Traits,
			agent.Style,
			agent.Influences,
			agentNode.GetP2PNode(),
		)

		// Register on the agent's node
		registry.RegisterValidator(chainID, agent.ID, validatorInstance)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent role"})
		return
	}

	communication.BroadcastEvent(communication.EventAgentRegistered, agent)

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent registered successfully",
		"agentID": agent.ID,
		"p2pPort": newPort,
		"apiPort": agentNode.GetAPIPort(),
	})
}

// GetBlock - Fetch a block by height
func GetBlock(c *gin.Context) {
	chainID := c.GetString("chainID")
	height, err := strconv.Atoi(c.Param("height"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block height"})
		return
	}

	chain := core.GetChain(chainID)
	if chain == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	if height < 0 || height >= len(chain.Blocks) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}
	block := chain.Blocks[height]

	c.JSON(http.StatusOK, gin.H{"block": block})
}

// GetNetworkStatus - Returns the current status of ChaosChain
func GetNetworkStatus(c *gin.Context) {
	chainID := c.GetString("chainID")
	bc := core.GetChain(chainID)
	if bc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	// Get node count for this chain
	bc.NodesMu.RLock()
	nodeCount := len(bc.Nodes)
	bc.NodesMu.RUnlock()

	status := map[string]interface{}{
		"height":     len(bc.Blocks) - 1,
		"latestHash": bc.Blocks[len(bc.Blocks)-1].Hash(),
		"totalTxs":   len(bc.Blocks[len(bc.Blocks)-1].Txs),
		"nodeCount":  nodeCount,
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// SubmitTransaction - Allows an agent to submit a transaction
func SubmitTransaction(c *gin.Context) {
	chainID := c.GetString("chainID")

	var tx core.Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction format"})
		return
	}

	// Set the chainID on the transaction
	tx.ChainID = chainID

	// In production, you would get the private key from secure storage
	privateKey, err := core.GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate key"})
		return
	}

	// Sign the transaction
	if err := tx.SignTransaction(privateKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign transaction"})
		return
	}

	// Process the signed transaction
	bc := core.GetChain(chainID)
	if bc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	// Get the chain's mempool
	mp := mempool.GetMempool(chainID)
	if mp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mempool not found for chain"})
		return
	}

	if err := bc.ProcessTransaction(tx, mp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed: " + err.Error()})
		return
	}

	communication.BroadcastEvent(communication.EventNewTransaction, tx)

	c.JSON(http.StatusOK, gin.H{"message": "Transaction submitted successfully"})
}

// GetValidators - Returns the list of registered validators
func GetValidators(c *gin.Context) {
	chainID := c.GetString("chainID")
	validatorsList := validator.GetAllValidators(chainID)
	c.JSON(http.StatusOK, gin.H{"validators": validatorsList})
}

// GetSocialStatus - Retrieves an agent's social reputation
func GetSocialStatus(c *gin.Context) {
	agentID := c.Param("agentID")
	chainID := c.GetString("chainID")

	validator := validator.GetValidatorByID(chainID, agentID)
	if validator == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Validator not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agentID":       validator.ID,
		"name":          validator.Name,
		"mood":          validator.Mood,
		"relationships": validator.Relationships,
	})
}

// AddInfluence adds a new influence to a validator
func AddInfluence(c *gin.Context) {
	agentID := c.Param("agentID")
	chainID := c.GetString("chainID")
	var influence struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&influence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid influence data"})
		return
	}

	v := validator.GetValidatorByID(chainID, agentID)
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Validator not found"})
		return
	}

	v.Influences = append(v.Influences, influence.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Influence added successfully"})
}

// UpdateRelationship updates the relationship score between validators
func UpdateRelationship(c *gin.Context) {
	agentID := c.Param("agentID")
	chainID := c.GetString("chainID")
	var rel RelationshipUpdate
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship data"})
		return
	}
	rel.FromID = agentID // Set the from ID

	v := validator.GetValidatorByID(chainID, agentID)
	if v == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Validator not found"})
		return
	}

	// Validate score range
	if rel.Score < -1.0 || rel.Score > 1.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Score must be between -1.0 and 1.0"})
		return
	}

	v.Relationships[rel.TargetID] = rel.Score
	communication.BroadcastEvent(communication.EventAgentAlliance, rel)
	c.JSON(http.StatusOK, gin.H{"message": "Relationship updated successfully"})
}

// Add a function to get the badgerDB instance for a chain
func getDBForChain(chainID string) *storage.DBStorage {
	db, err := storage.GetDBStorage("./data", chainID)
	if err != nil {
		log.Printf("Failed to get DB for chain %s: %v", chainID, err)
		return nil
	}
	return db
}

// ProposeBlock creates a new block and starts consensus
func ProposeBlock(c *gin.Context) {
	chainID := c.GetString("chainID")
	waitForConsensus := c.DefaultQuery("wait", "false") == "true"

	// Get badgerDB instance for this chain
	db := getDBForChain(chainID)

	bc := core.GetChain(chainID)
	if bc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	block, err := bc.CreateBlock()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Immediately create the discussion thread for visualization.
	// The thread ID is derived from the block's hash.
	threadID := block.Hash()
	producerName := "ProducerAgent" // Replace with the actual producer agent's name as needed.
	title := fmt.Sprintf("Block Proposal %s", threadID)
	communication.CreateThread(threadID, title, producerName)

	// Set up a subscription to capture discussions for this block
	mp := mempool.GetMempool(chainID)

	if mp != nil {
		// Subscribe to agent discussions via broker
		_, err := core.NatsBrokerInstance.Subscribe("AGENT_DISCUSSION", func(m *nats.Msg) {
			var discussion consensus.Discussion
			if err := json.Unmarshal(m.Data, &discussion); err != nil {
				log.Printf("Error unmarshalling discussion from NATS: %v", err)
				return
			}

			// Store vote information in mempool for later storage in EigenDA
			ephemeralVote := mempool.EphemeralVote{
				ID:           discussion.ID,
				AgentID:      discussion.ValidatorID,
				VoteDecision: discussion.Type,
				Timestamp:    discussion.Timestamp.Unix(),
			}
			mp.EphemeralVotes = append(mp.EphemeralVotes, ephemeralVote)

			// Also persist to badgerDB
			if db != nil {
				voteKey := fmt.Sprintf("vote:%s:%s", chainID, discussion.ID)
				if err := db.PutObject(voteKey, ephemeralVote); err != nil {
					log.Printf("Failed to save vote to badgerDB: %v", err)
				}
			}

			// Store agent identity if not already stored
			agentIdentitiesMutex.Lock()
			if _, exists := mp.EphemeralAgentIdentities[discussion.ValidatorID]; !exists {
				// Get validator name if available
				v := validator.GetValidatorByID(chainID, discussion.ValidatorID)
				var agentName string
				if v != nil {
					agentName = v.Name
					mp.EphemeralAgentIdentities[discussion.ValidatorID] = v.Name
				} else {
					agentName = discussion.ValidatorID
					mp.EphemeralAgentIdentities[discussion.ValidatorID] = discussion.ValidatorID
				}

				// Also persist to badgerDB
				if db != nil {
					agentKey := fmt.Sprintf("agent:%s:%s", chainID, discussion.ValidatorID)
					if err := db.Put(agentKey, []byte(agentName)); err != nil {
						log.Printf("Failed to save agent identity to badgerDB: %v", err)
					}
				}
			}
			agentIdentitiesMutex.Unlock()
		})
		if err != nil {
			log.Printf("Error subscribing to AGENT_DISCUSSION: %v", err)
		}

		// Also subscribe to final votes
		_, err = core.NatsBrokerInstance.Subscribe("AGENT_VOTE", func(m *nats.Msg) {
			var vote consensus.Discussion
			if err := json.Unmarshal(m.Data, &vote); err != nil {
				log.Printf("Error unmarshalling vote from NATS: %v", err)
				return
			}

			// Store vote information in mempool for later storage in EigenDA
			ephemeralVote := mempool.EphemeralVote{
				AgentID:      vote.ValidatorID,
				VoteDecision: vote.Type,
				Timestamp:    vote.Timestamp.Unix(),
			}
			mp.EphemeralVotes = append(mp.EphemeralVotes, ephemeralVote)

			// Also persist to badgerDB
			if db != nil {
				voteKey := fmt.Sprintf("vote:%s:%s", chainID, vote.ID)
				if err := db.PutObject(voteKey, ephemeralVote); err != nil {
					log.Printf("Failed to save vote to badgerDB: %v", err)
				}
			}

			// Store agent identity if not already stored
			agentIdentitiesMutex.Lock()
			if _, exists := mp.EphemeralAgentIdentities[vote.ValidatorID]; !exists {
				// Get validator name if available
				v := validator.GetValidatorByID(chainID, vote.ValidatorID)
				var agentName string
				if v != nil {
					agentName = v.Name
					mp.EphemeralAgentIdentities[vote.ValidatorID] = v.Name
				} else {
					agentName = vote.ValidatorID
					mp.EphemeralAgentIdentities[vote.ValidatorID] = vote.ValidatorID
				}

				// Also persist to badgerDB
				if db != nil {
					agentKey := fmt.Sprintf("agent:%s:%s", chainID, vote.ValidatorID)
					if err := db.Put(agentKey, []byte(agentName)); err != nil {
						log.Printf("Failed to save agent identity to badgerDB: %v", err)
					}
				}
			}
			agentIdentitiesMutex.Unlock()
		})
		if err != nil {
			log.Printf("Error subscribing to AGENT_VOTE: %v", err)
		}
	}

	cm := consensus.GetConsensusManager(chainID)
	if err := cm.ProposeBlock(block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to start consensus: " + err.Error()})
		return
	}

	if !waitForConsensus {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Block proposed successfully, consensus started",
			"block":     block,
			"thread_id": threadID,
		})
		return
	}

	result := make(chan consensus.ConsensusResult)
	cm.SubscribeResult(int64(block.Height), result)

	// Calculate total expected time: all rounds + voting round + buffer + safety margin
	totalTime := time.Duration(consensus.DiscussionRounds+1)*consensus.RoundDuration +
		5*time.Second + // Buffer time
		2*time.Second // Safety margin

	select {
	case consensusResult := <-result:
		// Store offchain data to EigenDA and clear temporary mempool data
		if mp := mempool.GetMempool(chainID); mp != nil {
			// Get discussions from consensus if available
			var discussions []consensus.Discussion
			activeConsensus := cm.GetActiveConsensus()
			if activeConsensus != nil {
				discussions = activeConsensus.GetDiscussions()
			}

			votes := make([]da.Vote, len(mp.EphemeralVotes))
			for i, ev := range mp.EphemeralVotes {
				votes[i] = da.Vote{
					AgentID:      ev.AgentID,
					VoteDecision: ev.VoteDecision,
					Timestamp:    ev.Timestamp,
				}
			}

			offchain := da.OffchainData{
				ChainID:     chainID,
				BlockHash:   threadID,
				BlockHeight: block.Height,
				Discussions: discussions,
				Votes:       votes,
				Outcome: func() string {
					if consensusResult.State == consensus.Accepted {
						return "accepted"
					}
					return "rejected"
				}(),
				AgentIdentities: mp.EphemeralAgentIdentities,
				Timestamp:       time.Now().Unix(),
			}

			// Save to EigenDA
			if id, err := da.SaveOffchainData(offchain); err != nil {
				log.Printf("Error saving offchain data: %v", err)
			} else {
				log.Printf("Offchain data saved with id: %s", id)

				// Also save the blob reference to badgerDB
				if db != nil {
					blobRefKey := fmt.Sprintf("blobref:%s:%s", chainID, threadID)
					blobRef := struct {
						BlobID      string
						BlockHash   string
						BlockHeight int
						Timestamp   int64
					}{
						BlobID:      id,
						BlockHash:   threadID,
						BlockHeight: block.Height,
						Timestamp:   time.Now().Unix(),
					}
					if err := db.PutObject(blobRefKey, blobRef); err != nil {
						log.Printf("Failed to save blob reference to badgerDB: %v", err)
					}
				}
			}

			// Clear temporary data from mempool
			mp.ClearTemporaryData()

			// Also clear temporary data from badgerDB
			if db != nil {
				prefixes := []string{
					fmt.Sprintf("vote:%s:", chainID),
					fmt.Sprintf("agent:%s:", chainID),
				}
				for _, prefix := range prefixes {
					if err := db.DeleteByPrefix(prefix); err != nil {
						log.Printf("Failed to clear data with prefix %s from badgerDB: %v", prefix, err)
					}
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Consensus completed",
			"block":     block,
			"accepted":  consensusResult.State == consensus.Accepted,
			"support":   consensusResult.Support,
			"oppose":    consensusResult.Oppose,
			"thread_id": threadID,
		})
	case <-time.After(totalTime):
		// Close the broker to clean up subscriptions

		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error":     "Consensus timed out",
			"block":     block,
			"thread_id": threadID,
		})
	}
}

// GetAllThreads returns all active discussion threads for monitoring.
func GetAllThreads(c *gin.Context) {
	threads := communication.GetAllThreads() // We'll implement this function in forum
	c.JSON(http.StatusOK, threads)
}

type CreateChainRequest struct {
	ChainID string `json:"chain_id" binding:"required"`
}

// CreateChain creates a new blockchain instance
func CreateChain(c *gin.Context) {
	var req CreateChainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Check if chain already exists
	if core.GetChain(req.ChainID) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Chain already exists"})
		return
	}

	// Find available ports for the bootstrap node
	p2pPort := findAvailablePort()
	apiPort := findAvailableAPIPort()

	// Create bootstrap node for the new chain
	bootstrapNode := node.NewNode(node.NodeConfig{
		ChainConfig: p2p.ChainConfig{
			ChainID: req.ChainID,
			P2PPort: p2pPort,
			APIPort: apiPort,
		},
		// No bootstrap node address as this will be the bootstrap node
	})

	if err := bootstrapNode.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start bootstrap node"})
		return
	}

	// Initialize new chain with its own mempool
	mp := mempool.NewMempool(req.ChainID)
	core.InitBlockchain(req.ChainID, mp)

	// Register the bootstrap node with the chain
	chain := core.GetChain(req.ChainID)
	addr := fmt.Sprintf("localhost:%d", p2pPort)
	chain.RegisterNode(addr, bootstrapNode.GetP2PNode())

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Chain created successfully",
		"chain_id": req.ChainID,
		"bootstrap_node": map[string]int{
			"p2p_port": p2pPort,
			"api_port": apiPort,
		},
	})
}

// ListChains returns all available chains
func ListChains(c *gin.Context) {
	chains := core.GetAllChains()
	c.JSON(http.StatusOK, gin.H{
		"chains": chains,
	})
}

// GetBlockDiscussions returns the discussions for a specific block by hash
func GetBlockDiscussions(c *gin.Context) {
	chainID := c.GetString("chainID")
	blockHash := c.Param("blockHash")

	// Try to get the blob reference from badgerDB first
	db := getDBForChain(chainID)
	if db != nil {
		blobRefKey := fmt.Sprintf("blobref:%s:%s", chainID, blockHash)
		var blobRef struct {
			BlobID      string
			BlockHash   string
			BlockHeight int
			Timestamp   int64
		}

		if err := db.GetObject(blobRefKey, &blobRef); err == nil {
			// Found in badgerDB, retrieve from EigenDA
			offchainData, err := da.GetOffchainData(blobRef.BlobID)
			if err == nil {
				// Format timestamps for better readability in the response
				formattedDiscussions := make([]map[string]interface{}, len(offchainData.Discussions))
				for i, d := range offchainData.Discussions {
					formattedDiscussions[i] = map[string]interface{}{
						"id":          d.ID,
						"validatorId": d.ValidatorID,
						"message":     d.Message,
						"timestamp":   d.Timestamp.Format(time.RFC3339),
						"type":        d.Type,
						"round":       d.Round,
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"blockHash":   blockHash,
					"blockHeight": blobRef.BlockHeight,
					"discussions": formattedDiscussions,
					"votes":       offchainData.Votes,
					"outcome":     offchainData.Outcome,
					"agents":      offchainData.AgentIdentities,
					"timestamp":   time.Unix(offchainData.Timestamp, 0).Format(time.RFC3339),
				})
				return
			}
		}
	}

	// Fall back to the original method if not found in badgerDB
	ref, found := da.GetBlobReferenceByBlockHash(chainID, blockHash)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No discussions found for this block"})
		return
	}

	// Retrieve the data from EigenDA
	offchainData, err := da.GetOffchainData(ref.BlobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve discussions: %v", err)})
		return
	}

	// Format timestamps for better readability in the response
	formattedDiscussions := make([]map[string]interface{}, len(offchainData.Discussions))
	for i, d := range offchainData.Discussions {
		formattedDiscussions[i] = map[string]interface{}{
			"id":          d.ID,
			"validatorId": d.ValidatorID,
			"message":     d.Message,
			"timestamp":   d.Timestamp.Format(time.RFC3339),
			"type":        d.Type,
			"round":       d.Round,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"blockHash":   blockHash,
		"blockHeight": ref.BlockHeight,
		"discussions": formattedDiscussions,
		"votes":       offchainData.Votes,
		"outcome":     offchainData.Outcome,
		"agents":      offchainData.AgentIdentities,
		"timestamp":   time.Unix(offchainData.Timestamp, 0).Format(time.RFC3339),
	})
}

// GetBlockDiscussionsByHeight returns the discussions for a specific block by height
func GetBlockDiscussionsByHeight(c *gin.Context) {
	chainID := c.GetString("chainID")
	heightStr := c.Param("height")

	height, err := strconv.Atoi(heightStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block height"})
		return
	}

	// Try to get the blob reference from badgerDB first
	db := getDBForChain(chainID)
	if db != nil {
		// Get all blob references and find the one with matching height
		blobRefs, err := db.GetByPrefix(fmt.Sprintf("blobref:%s:", chainID))
		if err == nil {
			for _, data := range blobRefs {
				var blobRef struct {
					BlobID      string
					BlockHash   string
					BlockHeight int
					Timestamp   int64
				}

				if err := json.Unmarshal(data, &blobRef); err == nil {
					if blobRef.BlockHeight == height {
						// Found in badgerDB, retrieve from EigenDA
						offchainData, err := da.GetOffchainData(blobRef.BlobID)
						if err == nil {
							c.JSON(http.StatusOK, gin.H{
								"blockHash":   blobRef.BlockHash,
								"blockHeight": height,
								"discussions": offchainData.Discussions,
								"votes":       offchainData.Votes,
								"outcome":     offchainData.Outcome,
								"agents":      offchainData.AgentIdentities,
								"timestamp":   offchainData.Timestamp,
							})
							return
						}
					}
				}
			}
		}
	}

	// Fall back to the original method if not found in badgerDB
	ref, found := da.GetBlobReferenceByHeight(chainID, height)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "No discussions found for this block height"})
		return
	}

	// Retrieve the data from EigenDA
	offchainData, err := da.GetOffchainData(ref.BlobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve discussions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"blockHash":   ref.BlockHash,
		"blockHeight": height,
		"discussions": offchainData.Discussions,
		"votes":       offchainData.Votes,
		"outcome":     offchainData.Outcome,
		"agents":      offchainData.AgentIdentities,
		"timestamp":   offchainData.Timestamp,
	})
}

// ListBlockDiscussions returns a list of all blocks with discussions for a chain
func ListBlockDiscussions(c *gin.Context) {
	chainID := c.GetString("chainID")

	// Try to get the blob references from badgerDB first
	db := getDBForChain(chainID)
	if db != nil {
		blobRefs, err := db.GetByPrefix(fmt.Sprintf("blobref:%s:", chainID))
		if err == nil && len(blobRefs) > 0 {
			blocks := make([]map[string]interface{}, 0, len(blobRefs))

			for _, data := range blobRefs {
				var blobRef struct {
					BlobID      string
					BlockHash   string
					BlockHeight int
					Timestamp   int64
				}

				if err := json.Unmarshal(data, &blobRef); err == nil {
					// Try to get outcome from EigenDA
					outcome := "unknown"
					offchainData, err := da.GetOffchainData(blobRef.BlobID)
					if err == nil {
						outcome = offchainData.Outcome
					}

					blocks = append(blocks, map[string]interface{}{
						"blockHash":   blobRef.BlockHash,
						"blockHeight": blobRef.BlockHeight,
						"outcome":     outcome,
						"timestamp":   blobRef.Timestamp,
						"blobId":      blobRef.BlobID,
					})
				}
			}

			if len(blocks) > 0 {
				c.JSON(http.StatusOK, gin.H{"blocks": blocks})
				return
			}
		}
	}

	// Fall back to the original method if not found in badgerDB
	refs := da.GetBlobReferencesForChain(chainID)
	if len(refs) == 0 {
		c.JSON(http.StatusOK, gin.H{"blocks": []interface{}{}})
		return
	}

	// Create a summary for each block
	blocks := make([]map[string]interface{}, len(refs))
	for i, ref := range refs {
		blocks[i] = map[string]interface{}{
			"blockHash":   ref.BlockHash,
			"blockHeight": ref.BlockHeight,
			"outcome":     ref.Outcome,
			"timestamp":   ref.Timestamp,
			"blobId":      ref.BlobID,
		}
	}

	c.JSON(http.StatusOK, gin.H{"blocks": blocks})
}
