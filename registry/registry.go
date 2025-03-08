package registry

import (
	"sync"

	"github.com/NethermindEth/chaoschain-launchpad/producer"
	"github.com/NethermindEth/chaoschain-launchpad/validator"
)

var (
	producers  = make(map[string]*producer.Producer)
	validators = make(map[string]*validator.Validator)
	agentLock  sync.Mutex
)

func RegisterProducer(id string, p *producer.Producer) {
	agentLock.Lock()
	defer agentLock.Unlock()
	producers[id] = p
}

func RegisterValidator(id string, v *validator.Validator) {
	agentLock.Lock()
	defer agentLock.Unlock()
	validators[id] = v
}
