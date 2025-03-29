package abci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
	types "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

type Application struct {
	chainID           string
	mu                sync.RWMutex
	discussions       map[string]map[string]bool
	validators        []types.ValidatorUpdate // Persistent validator set
	pendingValUpdates []types.ValidatorUpdate // Diffs to return in EndBlock
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
	// Use the validators from the genesis file
	app.validators = req.Validators

	// Log the validators we're using
	for i, val := range app.validators {
		log.Printf("Using validator %d: %v", i, val)
	}

	// For PoA, we need to ensure we have at least one validator
	if len(app.validators) == 0 {
		log.Printf("WARNING: No validators in genesis, consensus may not work properly")
	}

	// Log validators to debug
	log.Printf("InitChain with %d validators from genesis", len(app.validators))

	return types.ResponseInitChain{
		Validators: app.validators, // Return the validators from genesis
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
			// Add PoA specific parameters
			Version: &tmproto.VersionParams{
				App: 1,
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
	log.Printf("DeliverTx received: %X", req.Tx)

	// Decode transaction
	var tx core.Transaction
	if err := json.Unmarshal(req.Tx, &tx); err != nil {
		log.Printf("Failed to unmarshal transaction: %v", err)
		return types.ResponseDeliverTx{
			Code: 1,
			Log:  fmt.Sprintf("Invalid transaction format: %v", err),
		}
	}

	log.Printf("Processing transaction: %+v", tx)

	// Handle different transaction types
	switch tx.Type {
	case "register_validator":
		// This is a validator registration transaction
		if len(tx.Data) == 0 {
			return types.ResponseDeliverTx{
				Code: 1,
				Log:  "Missing validator public key",
			}
		}

		// Create public key from bytes
		pubKey := ed25519.PubKey(tx.Data)

		// Register the validator with voting power
		app.RegisterValidator(pubKey, 100) // Give it some voting power

		log.Printf("Registered validator %s with pubkey %X", tx.From, tx.Data)

		return types.ResponseDeliverTx{
			Code: 0,
			Log:  fmt.Sprintf("Validator %s registered successfully", tx.From),
		}

	default:
		// Handle other transaction types
		return types.ResponseDeliverTx{Code: 0}
	}
}

func (app *Application) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	return types.ResponseBeginBlock{}
}

func (app *Application) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	app.mu.Lock()
	defer app.mu.Unlock()

	log.Printf("EndBlock at height %d — %d new validator updates", req.Height, len(app.pendingValUpdates))

	// Log each validator update in detail
	for i, update := range app.pendingValUpdates {
		log.Printf("Validator update %d: pubkey=%X, power=%d",
			i, update.PubKey.GetEd25519(), update.Power)
	}

	updates := app.pendingValUpdates
	app.pendingValUpdates = nil // Clear for next block

	// Log the response we're returning
	log.Printf("Returning %d validator updates in EndBlock response", len(updates))

	return types.ResponseEndBlock{
		ValidatorUpdates: updates,
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
	// TODO: Implement PrepareProposal

	log.Printf("PrepareProposal called with %d transactions", len(req.Txs))

	app.mu.Lock()
	defer app.mu.Unlock()

	var validTxs [][]byte
	for _, tx := range req.Txs {
		// Decode transaction
		var transaction core.Transaction
		if err := json.Unmarshal(tx, &transaction); err != nil {
			continue
		}

		// Always include validator registration txs
		if transaction.Type == "register_validator" {
			log.Printf("Including validator registration tx from %s", transaction.From)
			validTxs = append(validTxs, tx)
			continue
		}

		log.Printf("PrepareProposal including %d txs", len(validTxs))

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
		// for _, relatedValidator := range validator.GetAllValidators(app.chainID) {
		// 	relationship := proposer.Relationships[relatedValidator.ID]
		// 	// If strongly influenced by a validator, consider their opinion
		// 	if relationship > 0.7 || relationship < -0.7 {
		// 		// Simulate related validator's opinion based on relationship
		// 		app.discussions[txHash][relatedValidator.ID] = relationship > 0
		// 	}
		// }

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

	// Always accept proposals during development
	return types.ResponseProcessProposal{Status: types.ResponseProcessProposal_ACCEPT}
}

// RegisterValidator adds a new validator to the set
func (app *Application) RegisterValidator(pubKey crypto.PubKey, power int64) {
	app.mu.Lock()
	defer app.mu.Unlock()

	valUpdate := types.Ed25519ValidatorUpdate(pubKey.Bytes(), power)

	// Log the validator being registered
	log.Printf("Registering validator with address: %X, power: %d", pubKey.Address(), power)

	// Check if validator already exists
	for _, val := range app.validators {
		if bytes.Equal(val.PubKey.GetEd25519(), pubKey.Bytes()) {
			// Already exists, no update needed
			log.Printf("Validator already exists, not adding again")
			return
		}
	}

	// Add to persistent set
	app.validators = append(app.validators, valUpdate)
	log.Printf("Added validator to persistent set, now have %d validators", len(app.validators))

	// Also include in the updates for EndBlock
	app.pendingValUpdates = append(app.pendingValUpdates, valUpdate)
	log.Printf("Added validator to pending updates, now have %d pending updates", len(app.pendingValUpdates))
}
