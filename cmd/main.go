package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/NethermindEth/chaoschain-launchpad/api"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	_ "github.com/NethermindEth/chaoschain-launchpad/config" // Initialize config
	"github.com/NethermindEth/chaoschain-launchpad/core"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line flags
	chainID := flag.String("chain", "mainnet", "Chain ID")
	p2pPort := flag.Int("p2p-port", 26656, "CometBFT P2P port")
	rpcPort := flag.Int("rpc-port", 26657, "CometBFT RPC port")
	nats := flag.String("nats", "nats://localhost:4222", "NATS URL")
	flag.Parse()

	// Create data directory for the chain
	dataDir := fmt.Sprintf("./data/%s", *chainID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Create CometBFT config
	config := cfg.DefaultConfig()
	config.BaseConfig.RootDir = dataDir
	config.Moniker = *chainID
	config.P2P.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", *p2pPort)
	config.RPC.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", *rpcPort)

	// Genesis node settings
	config.P2P.Seeds = "" // No seeds for genesis node
	config.P2P.PersistentPeers = ""

	genesisNode, err := node.NewNode(config, *chainID)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	err = genesisNode.Start(context.Background())
	if err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Start NATS messaging
	core.SetupNATS(*nats)
	defer core.CloseNATS()

	log.Printf("Genesis node for chain %s started with P2P port %d and RPC port %d",
		*chainID, *p2pPort, *rpcPort)

	// Start API server
	router := gin.New()
	api.SetupRoutes(router, *chainID)
	log.Fatal(router.Run(fmt.Sprintf(":%d", *rpcPort)))
}
