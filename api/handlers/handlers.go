package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NethermindEth/chaoschain-launchpad/ai"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/registry"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

// RegisterAgent - Registers a new AI agent (Producer or Validator)
func RegisterAgent(c *gin.Context) {
	var agent core.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}

	// Assign a unique ID to the agent
	agent.ID = uuid.New().String()

	if agent.Role == "producer" {
		// Create personality from agent data
		personality := ai.Personality{
			Name:   agent.Name,
			Traits: agent.Traits,
			Style:  agent.Style,
		}

		// Create producer with mempool and personality
		mp := mempool.GetMempool().(*mempool.Mempool)
		producerInstance := producer.NewProducer(mp, personality)

		// Register producer
		registry.RegisterProducer(agent.ID, producerInstance)

	} else if agent.Role == "validator" {
		// Ensure P2P node exists
		p2pNode := p2p.GetP2PNode() // Fetch the existing P2P node

		if p2pNode == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "P2P node not initialized"})
			return
		}

		// Create validator instance
		validatorInstance := validator.NewValidator(agent.ID, agent.Name, agent.Traits, agent.Style, agent.Influences, p2pNode)

		// Register validator
		registry.RegisterValidator(agent.ID, validatorInstance)

	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent registered successfully", "agentID": agent.ID})
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
