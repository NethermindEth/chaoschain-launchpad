package da

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/nats-io/nats.go"

	"github.com/Layr-Labs/eigenda/api/clients"
	"github.com/Layr-Labs/eigenda/core/auth"
	"github.com/Layr-Labs/eigenda/encoding/utils/codec"
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

// SetupGlobalDAService initializes the global DataAvailabilityService instance
func SetupGlobalDAService(natsURL string) error {
	globalDAServiceOnce.Do(func() {
		var service *DataAvailabilityService
		service, globalDAServiceErr = NewDataAvailabilityService(natsURL)
		if globalDAServiceErr != nil {
			log.Printf("Failed to initialize global DA service: %v", globalDAServiceErr)
			return
		}

		// Set up subscriptions for data events
		globalDAServiceErr = service.SetupSubscriptions(
			func(dataID string) { log.Printf("Data stored with ID: %s", dataID) },
			func(dataID string) { log.Printf("Data retrieved with ID: %s", dataID) },
		)
		if globalDAServiceErr != nil {
			log.Printf("Failed to set up DA service subscriptions: %v", globalDAServiceErr)
			return
		}

		GlobalDAService = service
		log.Println("Global EigenDA service initialized successfully")

		// Initialize the master index
		if err := InitializeMasterIndex(); err != nil {
			log.Printf("Failed to initialize master index: %v", err)
			globalDAServiceErr = err
			return
		}
		log.Println("Master index initialized successfully")
	})

	return globalDAServiceErr
}

// GetGlobalDAService returns the global DataAvailabilityService instance
func GetGlobalDAService() *DataAvailabilityService {
	if GlobalDAService == nil {
		log.Println("Warning: Global DA service not initialized")
	}
	return GlobalDAService
}

// CloseGlobalDAService closes the global DataAvailabilityService instance
func CloseGlobalDAService() {
	if GlobalDAService != nil {
		GlobalDAService.Close()
		GlobalDAService = nil
		log.Println("Global EigenDA service closed")
	}
}

// DataAvailabilityService handles interactions with EigenDA
type DataAvailabilityService struct {
	messenger *communication.Messenger
	client    clients.DisperserClient
}

// NewDataAvailabilityService creates a new DA service
func NewDataAvailabilityService(natsURL string) (*DataAvailabilityService, error) {
	messenger, err := communication.NewMessenger(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create messenger: %w", err)
	}

	// Get authentication key from environment
	eigendaAuthKey, ok := os.LookupEnv("EIGENDA_AUTH_PK")
	if !ok {
		return nil, fmt.Errorf("EIGENDA_AUTH_PK environment variable not set")
	}

	// Validate key length and remove optional '0x' prefix
	eigendaAuthKey = strings.TrimSpace(eigendaAuthKey)
	eigendaAuthKey = strings.TrimPrefix(eigendaAuthKey, "0x")
	eigendaAuthKey = strings.ReplaceAll(eigendaAuthKey, ".", "")
	if len(eigendaAuthKey) < 64 {
		eigendaAuthKey = strings.Repeat("0", 64-len(eigendaAuthKey)) + eigendaAuthKey
	} else if len(eigendaAuthKey) > 64 {
		return nil, fmt.Errorf("invalid EIGENDA_AUTH_PK length: got %d, expected 64 hex characters", len(eigendaAuthKey))
	}

	// Validate that the key is a valid hex string
	if _, err := hex.DecodeString(eigendaAuthKey); err != nil {
		return nil, fmt.Errorf("invalid EIGENDA_AUTH_PK: hex decoding failed: %w", err)
	}

	// Set up authentication with private key using decoded bytes
	signer := auth.NewLocalBlobRequestSigner("0x" + eigendaAuthKey)

	// Configuration for the disperser client
	config := &clients.Config{
		Hostname:          EIGENDA_HOST,
		Port:              EIGENDA_PORT,
		Timeout:           EIGENDA_REQUEST_TIMEOUT,
		UseSecureGrpcFlag: true, // should be true for production
	}

	// Create the disperser client
	client, err := clients.NewDisperserClient(config, signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create disperser client: %w", err)
	}

	service := &DataAvailabilityService{
		messenger: messenger,
		client:    client,
	}

	return service, nil
}

// Helper function for retries
func withRetries(fn func() (string, error)) (string, error) {
	var err error
	for i := 0; i < MAX_RETRIES; i++ {
		var result string
		result, err = fn()
		if err == nil {
			return result, nil
		}
		fmt.Printf("Attempt %d failed: %v\n", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return "", err
}

// StoreData stores data in EigenDA and publishes dataID to NATS
func (s *DataAvailabilityService) StoreData(data map[string]interface{}) (string, error) {
	if data == nil {
		return "", fmt.Errorf("data is required")
	}

	// Convert data to JSON bytes
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encode data to be compatible with bn254 field element constraints
	encodedData := codec.ConvertByPaddingEmptyByte(jsonData)

	// Add retry logic for dispersing the blob
	var dataID string
	err = retry(3, 2*time.Second, func() error {
		// Context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), EIGENDA_REQUEST_TIMEOUT)
		defer cancel()

		// Custom quorums (none for now, means we're dispersing to the default quorums)
		quorums := []uint8{}

		// Disperse the blob
		_, requestID, err := s.client.DisperseBlob(ctx, encodedData, quorums)
		if err != nil {
			return fmt.Errorf("error dispersing blob: %w", err)
		}

		// Convert requestID to string for use as dataID
		dataID = string(requestID)
		return nil
	})

	if err != nil {
		return "", err
	}

	// Wait for blob to be confirmed or finalized
	status, err := s.waitForBlobStatus(dataID)
	if err != nil {
		return dataID, fmt.Errorf("blob dispersed but status tracking failed: %w", err)
	}

	// Publish event using the messenger
	message := fmt.Sprintf(`{"dataID":"%s","status":"%s","timestamp":%d}`,
		dataID, status, time.Now().Unix())
	if err := s.messenger.PublishGlobal(SUBJECT_DATA_STORED, message); err != nil {
		return dataID, fmt.Errorf("data stored but failed to publish event: %w", err)
	}

	return dataID, nil
}

// Helper function for retries
func retry(attempts int, sleep time.Duration, f func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			log.Printf("Attempt %d failed: %v. Retrying in %v...", i+1, err, sleep)
			time.Sleep(sleep)
			// Exponential backoff
			sleep = sleep * 2
		}
	}
	return err
}

// waitForBlobStatus polls the blob status until it's finalized or failed
func (s *DataAvailabilityService) waitForBlobStatus(requestID string) (string, error) {
	// Create a context for the overall status checking
	statusOverallCtx, statusOverallCancel := context.WithTimeout(context.Background(), EIGENDA_MAX_WAIT_TIME)
	defer statusOverallCancel()

	ticker := time.NewTicker(EIGENDA_POLL_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Create a new context for each status request
			statusCtx, statusCancel := context.WithTimeout(statusOverallCtx, EIGENDA_REQUEST_TIMEOUT)

			// Get the blob status
			statusReply, err := s.client.GetBlobStatus(statusCtx, []byte(requestID))
			statusCancel()

			if err != nil {
				return "ERROR", fmt.Errorf("error getting blob status: %w", err)
			}

			// Check if the status is done
			status := statusReply.Status.String()
			if status == "FINALIZED" {
				fmt.Printf("Blob Status is finalized: %s\n", statusReply)
				return "FINALIZED", nil
			} else if status == "CONFIRMED" {
				fmt.Printf("Blob Status is confirmed: %s\n", statusReply)
				return "CONFIRMED", nil
			} else if status == "FAILED" {
				fmt.Printf("Blob Status is failed: %s\n", statusReply)
				return "FAILED", fmt.Errorf("blob dispersal failed with status: %v", status)
			}

			// Continue polling for other statuses
			fmt.Printf("Current Blob Status: %s\n", status)

		case <-statusOverallCtx.Done():
			return "TIMEOUT", fmt.Errorf("timed out waiting for blob to finalize")
		}
	}
}

// GetBlobStatus retrieves the current status of a blob from EigenDA
func (s *DataAvailabilityService) GetBlobStatus(dataID string) (interface{}, error) {
	if dataID == "" {
		return nil, fmt.Errorf("dataID is required")
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), EIGENDA_REQUEST_TIMEOUT)
	defer cancel()

	// Get the blob status using the client
	statusReply, err := s.client.GetBlobStatus(ctx, []byte(dataID))
	if err != nil {
		return nil, fmt.Errorf("error getting blob status: %w", err)
	}

	return statusReply, nil
}

// RetrieveData retrieves data from EigenDA using dataID
func (s *DataAvailabilityService) RetrieveData(dataID string) (map[string]interface{}, error) {
	if dataID == "" {
		return nil, fmt.Errorf("dataID is required")
	}

	// Create a context with timeout for retrieval
	ctx, cancel := context.WithTimeout(context.Background(), EIGENDA_REQUEST_TIMEOUT)
	defer cancel()

	// Retrieve the blob from the disperser
	blobData, err := s.retrieveBlobFromDisperser(ctx, dataID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blob: %w", err)
	}

	// Remove null bytes padding from the data
	decodedData := s.removeNullBytesPadding(blobData)

	// Log the retrieved data for debugging
	log.Printf("Retrieved data (length: %d): %s", len(decodedData), string(decodedData))

	// Check if data is empty
	if len(decodedData) == 0 {
		return nil, fmt.Errorf("retrieved data is empty after removing null bytes")
	}

	// Unmarshal the JSON data
	var result map[string]interface{}
	if err := json.Unmarshal(decodedData, &result); err != nil {
		// Try to decode using codec if standard unmarshal fails
		decodedBytes := codec.RemoveEmptyByteFromPaddedBytes(blobData)
		if len(decodedBytes) > 0 {
			if err := json.Unmarshal(decodedBytes, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal retrieved data: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to unmarshal retrieved data: %w", err)
		}
	}

	// Publish event that data was retrieved
	retrieveMsg := fmt.Sprintf(`{"dataID":"%s","timestamp":%d}`, dataID, time.Now().Unix())
	s.messenger.PublishGlobal(SUBJECT_DATA_RETRIEVED, retrieveMsg)

	return result, nil
}

// retrieveBlobFromDisperser retrieves a blob from EigenDA using the disperser client
func (s *DataAvailabilityService) retrieveBlobFromDisperser(ctx context.Context, dataID string) ([]byte, error) {
	// First, get the blob status to get the batch information needed for retrieval
	statusReply, err := s.client.GetBlobStatus(ctx, []byte(dataID))
	if err != nil {
		return nil, fmt.Errorf("failed to get blob status for retrieval: %w", err)
	}

	// Check if we have the necessary information for retrieval
	if statusReply.Info == nil || statusReply.Info.BlobVerificationProof == nil {
		return nil, fmt.Errorf("blob status doesn't contain verification proof needed for retrieval")
	}

	// Extract the required parameters from the status reply
	batchHeaderHash := statusReply.Info.BlobVerificationProof.BatchMetadata.BatchHeaderHash
	blobIndex := statusReply.Info.BlobVerificationProof.BlobIndex

	// Log the retrieval parameters for debugging
	log.Printf("Retrieving blob with batch header hash: %x, blob index: %d",
		batchHeaderHash, blobIndex)

	// Use the client's RetrieveBlob method with the correct parameters
	data, err := s.client.RetrieveBlob(ctx, batchHeaderHash, uint32(blobIndex))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve blob: %w", err)
	}

	return data, nil
}

// removeNullBytesPadding removes null bytes padding from the end of the data
func (s *DataAvailabilityService) removeNullBytesPadding(data []byte) []byte {
	// Find the first non-null byte from the beginning
	var startPos int
	for startPos = 0; startPos < len(data); startPos++ {
		if data[startPos] != 0 {
			break
		}
	}

	// Find the last non-null byte from the end
	var endPos int
	for endPos = len(data) - 1; endPos >= 0; endPos-- {
		if data[endPos] != 0 {
			break
		}
	}

	// If the data is all null bytes, return empty
	if startPos > endPos {
		return []byte{}
	}

	// Return the data between the first and last non-null bytes
	return data[startPos : endPos+1]
}

// SetupSubscriptions sets up NATS subscriptions for DA events
func (s *DataAvailabilityService) SetupSubscriptions(dataStoredHandler, dataRetrievedHandler func(dataID string)) error {
	// Subscribe to data stored events
	if dataStoredHandler != nil {
		err := s.messenger.SubscribeGlobal(SUBJECT_DATA_STORED, func(msg *nats.Msg) {
			var data map[string]interface{}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				fmt.Printf("Error parsing data stored event: %v\n", err)
				return
			}

			if dataID, ok := data["dataID"].(string); ok {
				dataStoredHandler(dataID)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to data stored events: %w", err)
		}
	}

	// Subscribe to data retrieved events
	if dataRetrievedHandler != nil {
		err := s.messenger.SubscribeGlobal(SUBJECT_DATA_RETRIEVED, func(msg *nats.Msg) {
			var data map[string]interface{}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				fmt.Printf("Error parsing data retrieved event: %v\n", err)
				return
			}

			if dataID, ok := data["dataID"].(string); ok {
				dataRetrievedHandler(dataID)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to data retrieved events: %w", err)
		}
	}

	return nil
}

// Close closes the messenger connection and client
func (s *DataAvailabilityService) Close() {
	// if s.messenger != nil {
	// 	s.messenger.Close()
	// }

	// if s.client != nil {
	// 	s.client.Close()
	// }
}

// Example usage
// func main() {
// 	data := map[string]interface{}{"message": "Hello EigenDA with NATS!"}
// 	dataID, err := storeDataInEigenDA(data)
// 	if err != nil {
// 		fmt.Println("Error storing data:", err)
// 		return
// 	}
// 	fmt.Println("Stored Data ID:", dataID)

// 	retrievedData, err := retrieveDataFromEigenDA(dataID)
// 	if err != nil {
// 		fmt.Println("Error retrieving data:", err)
// 		return
// 	}
// 	fmt.Println("Retrieved Data:", retrievedData)
// }
