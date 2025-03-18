package abci

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
	types "github.com/cometbft/cometbft/abci/types"
)

type Application struct {
	chainID string
	mu      sync.RWMutex
	// Track discussion results for each transaction
	discussions map[string]map[string]bool // txHash -> validatorID -> support
}

func NewApplication(chainID string) types.Application {
	return &Application{
		chainID:     chainID,
		discussions: make(map[string]map[string]bool),
	}
}

// Required ABCI methods
func (app *Application) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{
		Data:             "ChaosChain L2",
		Version:          "1.0.0",
		AppVersion:       1,
		LastBlockHeight:  0,
		LastBlockAppHash: []byte{},
	}
}

func (app *Application) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	return types.ResponseInitChain{}
}

func (app *Application) Query(req types.RequestQuery) types.ResponseQuery {
	return types.ResponseQuery{}
}

func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	return types.ResponseCheckTx{Code: 0}
}

func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	return types.ResponseDeliverTx{Code: 0}
}

func (app *Application) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	return types.ResponseBeginBlock{}
}

func (app *Application) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{}
}

func (app *Application) Commit() types.ResponseCommit {
	return types.ResponseCommit{}
}

func (app *Application) ListSnapshots(req types.RequestListSnapshots) types.ResponseListSnapshots {
	return types.ResponseListSnapshots{}
}

func (app *Application) OfferSnapshot(req types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	return types.ResponseOfferSnapshot{}
}

func (app *Application) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	return types.ResponseLoadSnapshotChunk{}
}

func (app *Application) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	return types.ResponseApplySnapshotChunk{}
}

// PrepareProposal is called when this validator is the proposer
func (app *Application) PrepareProposal(req types.RequestPrepareProposal) types.ResponsePrepareProposal {
	app.mu.Lock()
	defer app.mu.Unlock()

	var validTxs [][]byte
	for _, tx := range req.Txs {
		// Decode transaction
		var transaction core.Transaction
		if err := json.Unmarshal(tx, &transaction); err != nil {
			continue
		}

		// Get social validator info
		proposer := validator.GetSocialValidator(app.chainID, fmt.Sprintf("%X", req.ProposerAddress))
		if proposer == nil {
			continue
		}

		// Initialize discussion for this tx if not exists
		txHash := fmt.Sprintf("%x", tx)
		if _, exists := app.discussions[txHash]; !exists {
			app.discussions[txHash] = make(map[string]bool)
		}

		// AI agent (proposer) evaluates transaction based on relationships
		support := true // Default support
		for _, relatedValidator := range validator.GetAllValidators(app.chainID) {
			relationship := proposer.Relationships[relatedValidator.ID]
			// If strongly influenced by a validator, consider their opinion
			if relationship > 0.7 || relationship < -0.7 {
				// Simulate related validator's opinion based on relationship
				app.discussions[txHash][relatedValidator.ID] = relationship > 0
			}
		}

		// Record proposer's decision
		app.discussions[txHash][proposer.ID] = support

		// Add transaction if supported
		if support {
			validTxs = append(validTxs, tx)
		}
	}

	return types.ResponsePrepareProposal{Txs: validTxs}
}

// ProcessProposal is called on all other validators to validate the block proposal
func (app *Application) ProcessProposal(req types.RequestProcessProposal) types.ResponseProcessProposal {
	app.mu.Lock()
	defer app.mu.Unlock()

	validator := validator.GetSocialValidator(app.chainID, fmt.Sprintf("%X", req.ProposerAddress))
	if validator == nil {
		return types.ResponseProcessProposal{Status: types.ResponseProcessProposal_REJECT}
	}

	// Evaluate each transaction in the proposal
	for _, tx := range req.Txs {
		txHash := fmt.Sprintf("%x", tx)

		// Initialize discussion for this tx if not exists
		if _, exists := app.discussions[txHash]; !exists {
			app.discussions[txHash] = make(map[string]bool)
		}

		// Consider relationship with proposer
		relationship := validator.Relationships[fmt.Sprintf("%X", req.ProposerAddress)]
		if relationship < -0.5 {
			// Strongly negative relationship might lead to rejection
			return types.ResponseProcessProposal{Status: types.ResponseProcessProposal_REJECT}
		}
	}

	return types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}
}
