package core

import (
	"encoding/json"
)

// StateRoot represents the blockchain's state at a given block height
type StateRoot struct {
	StateID string            `json:"state_id"`
	Changes map[string]string `json:"changes"` // Stores arbitrary AI-generated changes
}

// ToJSON converts the state root to JSON
func (sr *StateRoot) ToJSON() string {
	jsonData, _ := json.Marshal(sr)
	return string(jsonData)
}
