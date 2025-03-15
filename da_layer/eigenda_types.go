package da

import (
	"sync"
	"time"

	"github.com/Layr-Labs/eigenda/api/clients"
	"github.com/NethermindEth/chaoschain-launchpad/communication"
)

const (
	// Updated EigenDA URLs for Holesky
	MAX_RETRIES = 3

	// NATS subjects
	SUBJECT_DATA_STORED    = "data.stored"
	SUBJECT_DATA_RETRIEVED = "data.retrieved"

	// EigenDA configuration
	EIGENDA_HOST            = "disperser-holesky.eigenda.xyz"
	EIGENDA_PORT            = "443"
	EIGENDA_REQUEST_TIMEOUT = 30 * time.Second
	EIGENDA_POLL_INTERVAL   = 5 * time.Second
	EIGENDA_MAX_WAIT_TIME   = 30 * time.Minute

	// EigenDA API endpoints
	EIGENDA_DISPERSE_URL = "https://disperser-holesky.eigenda.xyz:443/v1/blob"
	EIGENDA_STATUS_URL   = "https://disperser-holesky.eigenda.xyz:443/v1/blob/status"
	EIGENDA_RETRIEVE_URL = "https://disperser-holesky.eigenda.xyz:443/v1/blob"
)

// Global instance of the DataAvailabilityService
var (
	GlobalDAService     *DataAvailabilityService
	globalDAServiceOnce sync.Once
	globalDAServiceErr  error
)

// DataAvailabilityService handles interactions with EigenDA
type DataAvailabilityService struct {
	messenger *communication.Messenger
	client    clients.DisperserClient
}
