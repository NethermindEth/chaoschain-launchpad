package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/api"
	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	_ "github.com/NethermindEth/chaoschain-launchpad/config" // Initialize config
	"github.com/NethermindEth/chaoschain-launchpad/core"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/types"
	"github.com/gin-gonic/gin"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func main() {
	// Parse command line flags
	chainID := flag.String("chain", "mainnet", "Chain ID")
	p2pPort := flag.Int("p2p-port", 26656, "CometBFT P2P port")
	rpcPort := flag.Int("rpc-port", 26657, "CometBFT RPC port")
	apiPort := flag.Int("api-port", 8080, "API server port")
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

	// Initialize config files and validator keys
	if err := os.MkdirAll(config.BaseConfig.RootDir+"/config", 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
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
			log.Fatalf("Failed to generate node key: %v", err)
		}
	}

	// Initialize genesis.json if it doesn't exist
	genesisFile := config.GenesisFile()
	// Force regeneration of genesis file to ensure validators are included
	if err := os.Remove(genesisFile); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove existing genesis file: %v", err)
	}

	if !fileExists(genesisFile) {
		// Get the validator's public key
		privVal := privval.LoadFilePV(privValKeyFile, privValStateFile)
		pubKey, err := privVal.GetPubKey()
		if err != nil {
			log.Fatalf("Failed to get validator public key: %v", err)
		}

		// Create genesis validator directly
		genValidator := types.GenesisValidator{
			PubKey: pubKey,
			Power:  1000000, // Increase validator power significantly
			Name:   "genesis",
		}

		genDoc := types.GenesisDoc{
			ChainID:         *chainID,
			GenesisTime:     time.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
			Validators:      []types.GenesisValidator{genValidator},
		}

		// Validate genesis doc before saving
		if err := genDoc.ValidateAndComplete(); err != nil {
			log.Fatalf("Failed to validate genesis doc: %v", err)
		}

		// Ensure the validator was correctly added
		if len(genDoc.Validators) == 0 {
			log.Fatalf("No validators in genesis document after validation")
		}

		if err := genDoc.SaveAs(genesisFile); err != nil {
			log.Fatalf("Failed to create genesis file: %v", err)
		}
	}

	// Genesis node settings
	config.P2P.Seeds = "" // No seeds for genesis node
	config.P2P.PersistentPeers = ""

	genesisNode, err := node.NewNode(config, *chainID)
	if err != nil {
		log.Fatalf("main: Failed to create node: %v", err)
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
	log.Fatal(router.Run(fmt.Sprintf(":%d", *apiPort)))
}
