package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/consensus"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/registry"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

var (
	lastUsedPort = 8080
	portMutex    sync.Mutex
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
	var agent core.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}

	// Assign a unique ID
	agent.ID = uuid.New().String()

	// Get bootstrap node's P2P instance
	bootstrapNode := p2p.GetP2PNode()
	bootstrapPort := bootstrapNode.GetPort()
	if bootstrapPort == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bootstrap node not ready"})
		return
	}

	// Create a new node for this agent
	newPort := findAvailablePort()
	agentNode := node.NewNode(node.NodeConfig{
		P2PPort:       newPort,
		APIPort:       findAvailableAPIPort(),
		BootstrapNode: fmt.Sprintf("localhost:%d", bootstrapPort),
	})

	if err := agentNode.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start agent node"})
		return
	}

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
		registry.RegisterProducer(agent.ID, producerInstance)

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
		registry.RegisterValidator(agent.ID, validatorInstance)
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
	height, err := strconv.Atoi(c.Param("height"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block height"})
		return
	}

	block, exists := core.GetBlockByHeight(height)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"block": block})
}

// GetNetworkStatus - Returns the current status of ChaosChain
func GetNetworkStatus(c *gin.Context) {
	bc := core.GetBlockchain()
	status := map[string]interface{}{
		"height":     len(bc.Blocks) - 1,
		"latestHash": bc.Blocks[len(bc.Blocks)-1].Hash(),
		"totalTxs":   len(bc.Blocks[len(bc.Blocks)-1].Txs),
		"peerCount":  p2p.GetNetworkPeerCount(),
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// SubmitTransaction - Allows an agent to submit a transaction
func SubmitTransaction(c *gin.Context) {
	var tx core.Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction format"})
		return
	}

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
	bc := core.GetBlockchain()
	if err := bc.ProcessTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed: " + err.Error()})
		return
	}

	communication.BroadcastEvent(communication.EventNewTransaction, tx)

	c.JSON(http.StatusOK, gin.H{"message": "Transaction submitted successfully"})
}

// GetValidators - Returns the list of registered validators
func GetValidators(c *gin.Context) {
	validatorsList := validator.GetAllValidators()
	c.JSON(http.StatusOK, gin.H{"validators": validatorsList})
}

// GetSocialStatus - Retrieves an agent's social reputation
func GetSocialStatus(c *gin.Context) {
	agentID := c.Param("agentID")

	validator := validator.GetValidatorByID(agentID)
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
	var influence struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&influence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid influence data"})
		return
	}

	v := validator.GetValidatorByID(agentID)
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
	var rel RelationshipUpdate
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship data"})
		return
	}
	rel.FromID = agentID // Set the from ID

	v := validator.GetValidatorByID(agentID)
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

// ProposeBlock creates a new block and starts consensus
func ProposeBlock(c *gin.Context) {
	// Check if client wants to wait for consensus
	waitForConsensus := c.DefaultQuery("wait", "false") == "true"

	bc := core.GetBlockchain()
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

	cm := consensus.GetConsensusManager()
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
		c.JSON(http.StatusOK, gin.H{
			"message":   "Consensus completed",
			"block":     block,
			"accepted":  consensusResult.State == consensus.Accepted,
			"support":   consensusResult.Support,
			"oppose":    consensusResult.Oppose,
			"thread_id": threadID,
		})
	case <-time.After(totalTime):
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
