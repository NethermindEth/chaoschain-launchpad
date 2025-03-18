package abci

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
	types "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

type Application struct {
	chainID string
	mu      sync.RWMutex
	// Track discussion results for each transaction
	discussions map[string]map[string]bool // txHash -> validatorID -> support
	validators  []types.ValidatorUpdate    // Current validator set
}

func NewApplication(chainID string) types.Application {
	return &Application{
		chainID:     chainID,
		discussions: make(map[string]map[string]bool),
		validators:  make([]types.ValidatorUpdate, 0),
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
	log.Printf("InitChain called - Chain ID: %s", req.ChainId)
	log.Printf("InitChain request: %+v", req)
	log.Printf("InitChain Genesis Validators: %d", len(req.Validators))

	// Create a validator directly during InitChain
	// We need to manually create our validator since it's not being passed correctly
	valPubKey := types.Ed25519ValidatorUpdate(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		1000000)

	// Store validators from genesis
	app.validators = []types.ValidatorUpdate{valPubKey}

	// Log validators to debug
	log.Printf("InitChain with manually created validator")
	log.Printf("Created validator: %v", valPubKey)

	// Must return validators even if empty to properly initialize the validator set
	return types.ResponseInitChain{
		Validators: []types.ValidatorUpdate{valPubKey}, // Return our manually created validator
		ConsensusParams: &tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{
				MaxBytes: 22020096, // 21MB
				MaxGas:   -1,
			},
			Evidence: &tmproto.EvidenceParams{
				MaxAgeNumBlocks: 100000,
				MaxAgeDuration:  172800000000000, // 48 hours
				MaxBytes:        1048576,         // 1MB
			},
			Validator: &tmproto.ValidatorParams{
				PubKeyTypes: []string{"ed25519"},
			},
		},
	}
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
	// Genesis block is height 1
	if req.Height == 1 {
		log.Printf("EndBlock at height 1 - explicitly returning validators")

		// Load the validator key
		valPubKey := types.Ed25519ValidatorUpdate(
			[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			1000000)

		// Set this validator as active
		app.validators = []types.ValidatorUpdate{valPubKey}

		return types.ResponseEndBlock{
			ValidatorUpdates: []types.ValidatorUpdate{valPubKey},
		}
	}

	return types.ResponseEndBlock{
		ValidatorUpdates: app.validators,
	}
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
