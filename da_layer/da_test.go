package da

import (
	"fmt"
	"log"
	"os"
	"strings"
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

// TestEigenDARetrieval tests only the retrieval functionality
// To run this test: go test -v -run TestEigenDARetrieval ./da_layer
func TestEigenDARetrieval(t *testing.T) {
	// Skip in normal test runs
	if os.Getenv("RUN_EIGENDA_TEST") != "true" {
		t.Skip("Skipping EigenDA retrieval test. Set RUN_EIGENDA_TEST=true to run")
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

	// Use a known dataID from a previous successful test
	dataID := "fecf8d790c3a19d143a9fe87e5ca04e98f44a6bc189a5c94cfaf00007e81cc9d-313734313936343037323932383138393930312f302f33332f312f33332fe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Retrieve data
	fmt.Println("Retrieving data from EigenDA...")
	retrievedData, err := service.RetrieveData(dataID)
	if err != nil {
		t.Fatalf("Error retrieving data: %v", err)
	}

	// Print the retrieved data
	fmt.Printf("Retrieved data: %+v\n", retrievedData)

	// Verify that the data contains expected fields
	// if _, ok := retrievedData["chainId"]; !ok {
	// 	t.Errorf("Retrieved data missing 'chainId' field")
	// }

	// if _, ok := retrievedData["timestamp"]; !ok {
	// 	t.Errorf("Retrieved data missing 'timestamp' field")
	// }
}

// TestMain is used to set up any test dependencies
func TestMain(m *testing.M) {
	// Load environment variables from .env file
	loadEnvFile()

	// Set up NATS if needed for testing
	setupTestEnvironment()

	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}

// loadEnvFile loads environment variables from .env file in the project root
func loadEnvFile() {
	// Try to read .env file from project root
	envFile := "../.env"
	data, err := os.ReadFile(envFile)
	if err != nil {
		log.Printf("Warning: Could not read .env file: %v", err)
		return
	}

	// Parse each line and set environment variables
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			log.Printf("Set environment variable: %s", key)
		}
	}
}

// setupTestEnvironment ensures NATS is running for tests
func setupTestEnvironment() {
	// If we're running the EigenDA test, make sure NATS is available
	if os.Getenv("RUN_EIGENDA_TEST") == "true" {
		// You could start an embedded NATS server here if needed
		log.Println("Setting up test environment for EigenDA tests")
	}
}
