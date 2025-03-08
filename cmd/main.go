package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/NethermindEth/chaoschain-launchpad/api"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "P2P port")
	apiPort := flag.Int("api", 3000, "API port")
	bootstrapNode := flag.String("bootstrap", "", "Bootstrap node address")
	flag.Parse()

	// Create and start node
	node := node.NewNode(node.NodeConfig{
		P2PPort:       *port,
		APIPort:       *apiPort,
		BootstrapNode: *bootstrapNode,
	})

	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Start API server
	log.Printf("Starting API server on port %d...", *apiPort)
	router := gin.New()
	api.SetupRoutes(router)
	log.Fatal(router.Run(fmt.Sprintf(":%d", *apiPort)))
}
