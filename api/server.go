package api

import (
	"net/http"
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
	"github.com/gin-gonic/gin"
)

// Blockchain state
var (
	blockchain []producer.Block
	mutex      sync.Mutex
)

// StartServer initializes the REST API
func StartServer() {
	r := gin.Default()

	// Create a new block
	r.POST("/produce", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		data := c.PostForm("data")
		if len(blockchain) == 0 {
			// Genesis block
			genesisBlock := producer.NewBlock(0, "0", "Genesis Block")
			blockchain = append(blockchain, genesisBlock)
		}

		// Create a new block
		lastBlock := blockchain[len(blockchain)-1]
		newBlock := producer.NewBlock(lastBlock.Index+1, lastBlock.Hash, data)

		// Validate and append
		if validator.ValidateBlock(lastBlock, newBlock) {
			blockchain = append(blockchain, newBlock)
			c.JSON(http.StatusOK, gin.H{"message": "Block added!", "block": newBlock})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block"})
		}
	})

	// Fetch the blockchain
	r.GET("/chain", func(c *gin.Context) {
		c.JSON(http.StatusOK, blockchain)
	})

	r.Run(":8080") // Default port
}
