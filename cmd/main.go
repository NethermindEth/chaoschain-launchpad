package main

import (
	"log"

	"github.com/NethermindEth/chaoschain-launchpad/api"
	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/NethermindEth/chaoschain-launchpad/p2p"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize P2P node
	p2pNode := p2p.GetP2PNode()
	go func() {
		log.Printf("Starting P2P server on port 8080...")
		p2pNode.StartServer(8080)
	}()

	// Initialize mempool
	mempool.InitMempool(3600) // 1 hour expiration

	// Initialize blockchain with mempool
	core.InitBlockchain(mempool.GetMempool())

	// Start API server
	log.Printf("Starting API server on port 3000...")
	router := gin.New()
	api.SetupRoutes(router)
	log.Fatal(router.Run(":3000"))
}
