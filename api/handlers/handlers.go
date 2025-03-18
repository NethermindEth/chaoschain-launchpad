package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cometbft/cometbft/types"
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
	chainID := c.GetString("chainID")

	var agent core.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent data"})
		return
	}

	// Assign a unique ID
	agent.ID = uuid.New().String()

	// Create CometBFT config for the new node
	config := cfg.DefaultConfig()
	config.BaseConfig.RootDir = fmt.Sprintf("./data/%s", chainID)
	config.Moniker = fmt.Sprintf("%s-%s", agent.Role, agent.Name)

	// Set up P2P and RPC ports
	p2pPort := findAvailablePort()
	rpcPort := findAvailableAPIPort()
	config.P2P.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", p2pPort)
	config.RPC.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", rpcPort)

	// Connect to existing chain
	config.P2P.Seeds = fmt.Sprintf("tcp://localhost:26656") // Connect to genesis node

	// Create and start the node
	agentNode, err := node.NewNode(config, chainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create node: %v", err)})
		return
	}

	if err := agentNode.Start(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to start node: %v", err)})
		return
	}

	// Extract port from RPC address and register it
	rpcPort = extractPortFromAddress(config.RPC.ListenAddress)
	registerChainPort(chainID, rpcPort)

	// Store agent metadata (we'll need a new way to track agents)
	// TODO: Implement agent tracking

	communication.BroadcastEvent(communication.EventAgentRegistered, agent)

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent registered successfully",
		"agentID": agent.ID,
		"p2pPort": p2pPort,
		"rpcPort": rpcPort,
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

	// Connect to the specific chain's node using chainID
	rpcPort, err := getRPCPortForChain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Chain not found: %v", err)})
		return
	}

	client, err := rpchttp.New(fmt.Sprintf("tcp://localhost:%d", rpcPort), "/websocket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to node: %v", err)})
		return
	}

	// Verify we're connected to the right chain
	status, err := client.Status(context.Background())
	if err != nil || status.NodeInfo.Network != chainID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	// Get block at height
	heightPtr := new(int64)
	*heightPtr = int64(height)
	block, err := client.Block(context.Background(), heightPtr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Failed to get block: %v", err)})
		return
	}

	// Transform block data for response
	blockData := gin.H{
		"height":     block.Block.Height,
		"hash":       block.Block.Hash(),
		"timestamp":  block.Block.Time,
		"numTxs":     len(block.Block.Txs),
		"proposer":   block.Block.ProposerAddress,
		"validators": block.Block.LastCommit.Signatures,
	}

	c.JSON(http.StatusOK, gin.H{"block": blockData})
}

// GetNetworkStatus - Returns the current status of ChaosChain
func GetNetworkStatus(c *gin.Context) {
	chainID := c.GetString("chainID")

	// Get RPC port for this chain
	rpcPort, err := getRPCPortForChain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chain not found"})
		return
	}

	// Connect to the node
	client, err := rpchttp.New(fmt.Sprintf("tcp://localhost:%d", rpcPort), "/websocket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to node: %v", err)})
		return
	}

	// Get status from node
	status, err := client.Status(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get status: %v", err)})
		return
	}

	networkStatus := gin.H{
		"height":        status.SyncInfo.LatestBlockHeight,
		"latestHash":    status.SyncInfo.LatestBlockHash,
		"listenAddr":    status.NodeInfo.ListenAddr,
		"network":       status.NodeInfo.Network,
		"catchingUp":    status.SyncInfo.CatchingUp,
		"validatorInfo": status.ValidatorInfo,
	}

	c.JSON(http.StatusOK, gin.H{"status": networkStatus})
}

// SubmitTransaction - Allows an agent to submit a transaction
func SubmitTransaction(c *gin.Context) {
	chainID := c.GetString("chainID")

	var tx core.Transaction
	if err := c.ShouldBindJSON(&tx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction format"})
		return
	}

	// Get RPC port for this chain
	rpcPort, err := getRPCPortForChain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Chain not found: %v", err)})
		return
	}

	// Connect to the node
	client, err := rpchttp.New(fmt.Sprintf("tcp://localhost:%d", rpcPort), "/websocket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to node: %v", err)})
		return
	}

	// Encode transaction
	txBytes, err := tx.Marshal()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to encode transaction"})
		return
	}

	// Broadcast transaction
	result, err := client.BroadcastTxSync(context.Background(), txBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to broadcast tx: %v", err)})
		return
	}

	communication.BroadcastEvent(communication.EventNewTransaction, tx)

	c.JSON(http.StatusOK, gin.H{
		"message": "Transaction submitted successfully",
		"hash":    result.Hash.String(),
	})
}

// GetValidators - Returns the list of registered validators
func GetValidators(c *gin.Context) {
	chainID := c.GetString("chainID")

	// Get RPC port for this chain
	rpcPort, err := getRPCPortForChain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Chain not found: %v", err)})
		return
	}

	// Connect to the node
	client, err := rpchttp.New(fmt.Sprintf("tcp://localhost:%d", rpcPort), "/websocket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to node: %v", err)})
		return
	}

	// Get validators from CometBFT
	heightPtr := new(int64) // nil for latest height
	result, err := client.Validators(context.Background(), heightPtr, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get validators: %v", err)})
		return
	}

	// Transform validator data
	validators := make([]gin.H, len(result.Validators))
	for i, v := range result.Validators {
		validators[i] = gin.H{
			"address":          v.Address,
			"pubKey":           v.PubKey,
			"votingPower":      v.VotingPower,
			"proposerPriority": v.ProposerPriority,
		}
	}

	c.JSON(http.StatusOK, gin.H{"validators": validators})
}

// GetSocialStatus - Retrieves an agent's social reputation
func GetSocialStatus(c *gin.Context) {
	agentID := c.Param("agentID")
	chainID := c.GetString("chainID")

	// Get consensus validator info from CometBFT
	rpcPort, err := getRPCPortForChain(chainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Chain not found: %v", err)})
		return
	}

	client, err := rpchttp.New(fmt.Sprintf("tcp://localhost:%d", rpcPort), "/websocket")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect: %v", err)})
		return
	}

	// Verify validator exists in CometBFT
	result, err := client.Validators(context.Background(), nil, nil, nil)
	if err != nil || !validatorExists(result.Validators, agentID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Validator not found in consensus set"})
		return
	}

	// Get social info from our registry
	socialVal := validator.GetSocialValidator(chainID, agentID)
	if socialVal == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Validator not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agentID":       socialVal.ID,
		"name":          socialVal.Name,
		"mood":          socialVal.Mood,
		"relationships": socialVal.Relationships,
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

	// Check if chain already exists in our registry
	rpcPort, err := getRPCPortForChain(req.ChainID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Chain already exists"})
		return
	}

	// Find available ports for the bootstrap node
	p2pPort := findAvailablePort()
	rpcPort = findAvailableAPIPort()

	// Create CometBFT config for genesis node
	config := cfg.DefaultConfig()
	config.BaseConfig.RootDir = "./data/" + req.ChainID
	config.Moniker = "genesis-node"
	config.P2P.ListenAddress = "tcp://0.0.0.0:" + strconv.Itoa(p2pPort)
	config.RPC.ListenAddress = "tcp://0.0.0.0:" + strconv.Itoa(rpcPort)

	// Initialize config files and validator keys
	if err := os.MkdirAll(config.BaseConfig.RootDir+"/config", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create config directory: %v", err)})
		return
	}

	// Initialize validator key files
	privValKeyFile := config.PrivValidatorKeyFile()
	privValStateFile := config.PrivValidatorStateFile()
	if !fileExists(privValKeyFile) {
		privVal := privval.GenFilePV(privValKeyFile, privValStateFile)
		privVal.Save()
	}

	// Initialize node key file
	nodeKeyFile := config.NodeKeyFile()
	if !fileExists(nodeKeyFile) {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate node key: %v", err)})
			return
		}
	}

	// Initialize genesis.json if it doesn't exist
	genesisFile := config.GenesisFile()
	if !fileExists(genesisFile) {
		// Get the validator's public key
		privVal := privval.LoadFilePV(privValKeyFile, privValStateFile)
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get validator public key: %v", err)})
			return
		}

		// Create genesis validator directly
		genValidator := types.GenesisValidator{
			PubKey: pubKey,
			Power:  1000000, // Increase validator power significantly
			Name:   "genesis",
		}

		genDoc := types.GenesisDoc{
			ChainID:         req.ChainID,
			GenesisTime:     time.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
			Validators:      []types.GenesisValidator{genValidator},
		}

		// Validate genesis doc before saving
		if err := genDoc.ValidateAndComplete(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to validate genesis doc: %v", err)})
			return
		}

		if err := genDoc.SaveAs(genesisFile); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create genesis file: %v", err)})
			return
		}
	}

	// Create and start the genesis node
	genesisNode, err := node.NewNode(config, req.ChainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create genesis node: %v", err)})
		return
	}

	if err := genesisNode.Start(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start bootstrap node"})
		return
	}

	// Register chain in our registry
	registerChainPort(req.ChainID, rpcPort)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Chain created successfully",
		"chain_id": req.ChainID,
		"genesis_node": map[string]int{
			"p2p_port": p2pPort,
			"rpc_port": rpcPort,
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

func validatorExists(validators []*types.Validator, agentID string) bool {
	for _, v := range validators {
		if v.Address.String() == agentID {
			return true
		}
	}
	return false
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
