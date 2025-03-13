package da

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

// TestEigenDAIntegration is a manual test to verify EigenDA integration
// To run this test: go test -v -run TestEigenDAIntegration ./da_layer
func TestEigenDAIntegration(t *testing.T) {
	// Skip in normal test runs
	if os.Getenv("RUN_EIGENDA_TEST") != "true" {
		t.Skip("Skipping EigenDA integration test. Set RUN_EIGENDA_TEST=true to run")
	}

	// Check if auth key is set
	if os.Getenv("EIGENDA_AUTH_PK") == "" {
		t.Fatal("EIGENDA_AUTH_PK environment variable not set")
	}

	// Create DA service
	service, err := NewDataAvailabilityService("nats://localhost:4222")
	if err != nil {
		t.Fatalf("Failed to create DA service: %v", err)
	}
	defer service.Close()

	// Set up event tracking channels
	dataStoredCh := make(chan string, 1)
	dataRetrievedCh := make(chan string, 1)

	// Set up subscriptions
	err = service.SetupSubscriptions(
		func(dataID string) {
			fmt.Printf("Event received: Data stored with ID: %s\n", dataID)
			dataStoredCh <- dataID
		},
		func(dataID string) {
			fmt.Printf("Event received: Data retrieved with ID: %s\n", dataID)
			dataRetrievedCh <- dataID
		},
	)
	if err != nil {
		t.Fatalf("Failed to set up subscriptions: %v", err)
	}

	// Test data
	testData := map[string]interface{}{
		"message":   "Hello EigenDA with NATS!",
		"timestamp": time.Now().Unix(),
		"testId":    fmt.Sprintf("test-%d", time.Now().Unix()),
	}

	// Store data
	fmt.Println("Storing data in EigenDA...")
	dataID, err := service.StoreData(testData)
	if err != nil {
		t.Fatalf("Error storing data: %v", err)
	}
	fmt.Printf("Data stored with ID: %s\n", dataID)

	// Wait for stored event (with timeout)
	select {
	case receivedDataID := <-dataStoredCh:
		fmt.Printf("Received data stored event for ID: %s\n", receivedDataID)
	case <-time.After(2 * time.Minute):
		t.Fatalf("Timed out waiting for data stored event")
	}

	// Retrieve data
	fmt.Println("Retrieving data from EigenDA...")
	retrievedData, err := service.RetrieveData(dataID)
	if err != nil {
		t.Fatalf("Error retrieving data: %v", err)
	}

	// Verify retrieved data
	if retrievedData["message"] != testData["message"] {
		t.Errorf("Retrieved data doesn't match: got %v, want %v", 
			retrievedData["message"], testData["message"])
	}

	fmt.Printf("Retrieved data: %v\n", retrievedData)

	// Wait for retrieved event (with timeout)
	select {
	case <-dataRetrievedCh:
		fmt.Println("Received data retrieved event")
	case <-time.After(10 * time.Second):
		t.Fatalf("Timed out waiting for data retrieved event")
	}

	fmt.Println("EigenDA integration test completed successfully!")
}

// TestMain is used to set up any test dependencies
func TestMain(m *testing.M) {
	// Set up NATS if needed for testing
	setupTestEnvironment()
	
	// Run tests
	code := m.Run()
	
	// Clean up
	os.Exit(code)
}

// setupTestEnvironment ensures NATS is running for tests
func setupTestEnvironment() {
	// If we're running the EigenDA test, make sure NATS is available
	if os.Getenv("RUN_EIGENDA_TEST") == "true" {
		// You could start an embedded NATS server here if needed
		log.Println("Setting up test environment for EigenDA tests")
	}
} 