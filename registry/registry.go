package registry

import (
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

var (
	// Map of chainID -> map of producerID -> Producer
	producers = make(map[string]map[string]*producer.Producer)
	agentLock sync.Mutex
)

func RegisterProducer(chainID string, id string, p *producer.Producer) {
	agentLock.Lock()
	defer agentLock.Unlock()
	if producers[chainID] == nil {
		producers[chainID] = make(map[string]*producer.Producer)
	}
	producers[chainID][id] = p
}

func RegisterValidator(chainID string, id string, v *validator.Validator) {
	validator.RegisterValidator(chainID, id, v)
}
