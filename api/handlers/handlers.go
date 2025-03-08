package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
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
	status := core.GetNetworkStatus()
	c.JSON(http.StatusOK, gin.H{"status": status})
}

// SubmitTransaction - Allows an agent to submit a transaction
func SubmitTransaction(c *gin.Context) {
	var tx core.Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction format"})
		return
	}

	bc := core.GetBlockchain()
	if bc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Blockchain not initialized"})
		return
	}
	if err := bc.ProcessTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
		return
	}

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
	var rel struct {
		TargetID string  `json:"targetId"`
		Score    float64 `json:"score"` // -1.0 to 1.0
	}
	if err := c.ShouldBindJSON(&rel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relationship data"})
		return
	}

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
	c.JSON(http.StatusOK, gin.H{"message": "Relationship updated successfully"})
}
