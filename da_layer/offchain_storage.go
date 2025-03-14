package da

import (
	"encoding/json"
	"fmt"
)

// OffchainData represents the off-chain data stored in EigenDA for a specific chain.
type OffchainData struct {
	ChainID         string            `json:"chainId"`
	Discussions     []Discussion      `json:"discussions"`
	Votes           []Vote            `json:"votes"`
	Outcome         string            `json:"outcome"`
	AgentIdentities map[string]string `json:"agentIdentities"`
}

// Discussion represents an agent's discussion entry off-chain.
type Discussion struct {
	AgentID   string `json:"agentId"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Vote represents an agent's vote off-chain.
type Vote struct {
	AgentID      string `json:"agentId"`
	VoteDecision string `json:"voteDecision"`
	Timestamp    int64  `json:"timestamp"`
}

// SaveOffchainData stores off-chain data into EigenDA using the provided DataAvailabilityService.
// It marshals the off-chain data into a map and then stores it via StoreData.
func SaveOffchainData(svc *DataAvailabilityService, data OffchainData) (string, error) {
	// Update the timestamp if needed
	if data.Outcome == "" {
		data.Outcome = "pending"
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal offchain data: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return "", fmt.Errorf("failed to convert offchain data to map: %w", err)
	}

	return svc.StoreData(payload)
}

// Example usage:
// offchain := OffchainData{
//     ChainID: "chain-123",
//     Discussions: []Discussion{
//         {AgentID: "agent-1", Message: "Discussion message", Timestamp: time.Now().Unix()},
//     },
//     Votes: []Vote{
//         {AgentID: "agent-1", VoteDecision: "support", Timestamp: time.Now().Unix()},
//     },
//     Outcome: "pending",
//     AgentIdentities: map[string]string{
//         "agent-1": "Agent One",
//     },
// }
// id, err := SaveOffchainData(dataAvailabilityService, offchain)

// You can extend this module with retrieval and update functions as necessary.
