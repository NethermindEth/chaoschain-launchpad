package da

import (
	"context"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/communication"
	"github.com/nats-io/nats.go"

	"github.com/Layr-Labs/eigenda/clients"
	"github.com/Layr-Labs/eigenda/core/auth"
	"github.com/Layr-Labs/eigenda/encoding/utils/codec"
)

const (
	// Updated EigenDA URLs for Holesky
	MAX_RETRIES       = 3
	
	// NATS subjects
	SUBJECT_DATA_STORED = "data.stored"
	SUBJECT_DATA_RETRIEVED = "data.retrieved"
	
	// EigenDA configuration
	EIGENDA_HOST            = "disperser-holesky.eigenda.xyz"
	EIGENDA_PORT            = "443"
	EIGENDA_REQUEST_TIMEOUT = 10 * time.Second
	EIGENDA_POLL_INTERVAL   = 5 * time.Second
	EIGENDA_MAX_WAIT_TIME   = 30 * time.Minute
	
	// EigenDA API endpoints
	EIGENDA_DISPERSE_URL    = "https://disperser-holesky.eigenda.xyz:443/v1/blob"
	EIGENDA_STATUS_URL      = "https://disperser-holesky.eigenda.xyz:443/v1/blob/status"
	EIGENDA_RETRIEVE_URL    = "https://disperser-holesky.eigenda.xyz:443/v1/blob"
)

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
	signer := auth.NewSigner("0x" + eigendaAuthKey)

	// Configuration for the disperser client
	config := clients.NewConfig(
		EIGENDA_HOST,
		EIGENDA_PORT,
		EIGENDA_REQUEST_TIMEOUT,
		true, // useSecureGrpcFlag, should be true for production
	)

	// Create the disperser client
	client := clients.NewDisperserClient(config, signer)

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

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), EIGENDA_REQUEST_TIMEOUT)
	defer cancel()

	// Custom quorums (none for now, means we're dispersing to the default quorums)
	quorums := []uint8{}

	// Disperse the blob
	_, requestID, err := s.client.DisperseBlob(ctx, encodedData, quorums)
	if err != nil {
		return "", fmt.Errorf("error dispersing blob: %w", err)
	}

	// Convert requestID to string for use as dataID
	dataID := string(requestID)

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
				return "FINALIZED", nil
			} else if status == "CONFIRMED" {
				return "CONFIRMED", nil
			} else if status == "FAILED" {
				return "FAILED", fmt.Errorf("blob dispersal failed with status: %v", status)
			}
			
			// Continue polling for other statuses
			fmt.Printf("Current Blob Status: %s\n", status)
			
		case <-statusOverallCtx.Done():
			return "TIMEOUT", fmt.Errorf("timed out waiting for blob to finalize")
		}
	}
}

// RetrieveData retrieves data from EigenDA using dataID
func (s *DataAvailabilityService) RetrieveData(dataID string) (map[string]interface{}, error) {
	if dataID == "" {
		return nil, fmt.Errorf("dataID is required")
	}

	// Retrieve the blob using HTTP since the client doesn't have a RetrieveBlob method
	var result map[string]interface{}
	retrieveURL := fmt.Sprintf("https://%s:%s/v1/blob/%s", EIGENDA_HOST, EIGENDA_PORT, dataID)
	
	req, err := http.NewRequest("GET", retrieveURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create retrieve request: %w", err)
	}
	
	client := &http.Client{Timeout: EIGENDA_REQUEST_TIMEOUT}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send retrieve request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data, status code: %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read retrieve response: %w", err)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse retrieve response: %w", err)
	}
	
	// Extract and decode the data
	encodedData, ok := response["data"].([]byte)
	if !ok {
		// Try to get it as a string and convert
		if dataStr, ok := response["data"].(string); ok {
			encodedData = []byte(dataStr)
		} else {
			return nil, fmt.Errorf("failed to get data from response")
		}
	}
	
	// Since we can't find the exact function to remove padding, we'll handle it manually
	// This is a simple implementation to remove the empty byte padding
	var decodedData []byte
	for i := 0; i < len(encodedData); i++ {
		if i < len(encodedData)-1 && encodedData[i] == 0 && encodedData[i+1] == 0 {
			break
		}
		decodedData = append(decodedData, encodedData[i])
	}
	
	// Unmarshal the JSON data
	if err := json.Unmarshal(decodedData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal retrieved data: %w", err)
	}
	
	// Publish event that data was retrieved
	retrieveMsg := fmt.Sprintf(`{"dataID":"%s","timestamp":%d}`, dataID, time.Now().Unix())
	s.messenger.PublishGlobal(SUBJECT_DATA_RETRIEVED, retrieveMsg)
	
	return result, nil
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

// Close closes the messenger connection
func (s *DataAvailabilityService) Close() {
	// Any cleanup needed
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
