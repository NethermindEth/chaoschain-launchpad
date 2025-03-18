package node

import (
	"context"
	"fmt"
	"os"

	"github.com/NethermindEth/chaoschain-launchpad/consensus/abci"
	cfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"

	tmlog "github.com/cometbft/cometbft/libs/log"
)

type Node struct {
	cometCfg *cfg.Config
	node     *node.Node
	chainId  string
}

func NewNode(config *cfg.Config, chainId string) (*Node, error) {
	// Initialize config files and keys
	cfg.EnsureRoot(config.RootDir) // This function returns void, no need to check error

	// Create ABCI app
	app := abci.NewApplication(chainId)

	// Create node with default logger
	node, err := node.NewNode(
		config,
		privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile()),
		func() *p2p.NodeKey {
			nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
			if err != nil {
				panic(err) // Or handle error appropriately
			}
			return nodeKey
		}(),
		proxy.NewLocalClientCreator(app),
		node.DefaultGenesisDocProviderFunc(config),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(config.Instrumentation),
		tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %v", err)
	}

	return &Node{
		cometCfg: config,
		node:     node,
		chainId:  chainId,
	}, nil
}

func (n *Node) Start(ctx context.Context) error {
	if err := n.node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %v", err)
	}
	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	return n.node.Stop()
}
