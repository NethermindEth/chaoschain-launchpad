package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/NethermindEth/chaoschain-launchpad/cmd/node"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/p2p"
)

func main() {
	chainID := flag.String("chain", "", "Chain ID")
	agentID := flag.String("agent-id", "", "Agent ID")
	p2pPort := flag.Int("p2p-port", 0, "P2P Port")
	rpcPort := flag.Int("rpc-port", 0, "RPC Port")
	seedNode := flag.String("seed", "", "Seed node ID@IP:port")
	flag.Parse()

	if *chainID == "" || *agentID == "" {
		log.Fatal("chain-id and agent-id are required")
	}

	rootDir := fmt.Sprintf("./data/%s/%s", *chainID, *agentID)

	config := cfg.DefaultConfig()
	config.BaseConfig.RootDir = rootDir
	config.Moniker = *agentID
	config.P2P.ExternalAddress = fmt.Sprintf("tcp://127.0.0.1:%d", *p2pPort)
	config.P2P.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", *p2pPort)
	config.RPC.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%d", *rpcPort)

	config.P2P.SeedMode = false
	config.P2P.PexReactor = true
	config.P2P.AllowDuplicateIP = true
	config.P2P.AddrBookStrict = false

	// Get genesis node ID from its node_key.json if seed not provided
	if *seedNode == "" {
		genesisNodeKeyFile := fmt.Sprintf("./data/%s/genesis/config/node_key.json", *chainID)
		if _, err := os.Stat(genesisNodeKeyFile); err == nil {
			genesisNodeKey, err := p2p.LoadNodeKey(genesisNodeKeyFile)
			if err == nil {
				*seedNode = fmt.Sprintf("%s@127.0.0.1:26656", genesisNodeKey.ID())
				log.Printf("Using genesis node as seed: %s", *seedNode)
			}
		}
	}

	// Set seed node
	if *seedNode != "" {
		config.P2P.Seeds = *seedNode
		log.Printf("Using seed node: %s", *seedNode)
	}

	nodeInstance, err := node.NewNode(config, *chainID)
	if err != nil {
		log.Fatalf("Failed to create agent node: %v", err)
	}

	if err := nodeInstance.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start agent node: %v", err)
	}

	log.Printf("Agent node [%s] started on P2P %d, RPC %d", *agentID, *p2pPort, *rpcPort)
	select {} // keep process running
}
