# Data Availability Layer

This package provides a data availability layer using EigenDA for the ChaosChain Launchpad.

## Overview

The DA layer allows storing and retrieving data on EigenDA, a decentralized data availability network. It uses NATS for event broadcasting when data is stored or retrieved.

## Requirements

1. Go 1.21 or higher
2. NATS server running (default: localhost:4222)
3. EigenDA authentication private key

## Setup

### 1. Set Environment Variables

```bash
# Required: Your EigenDA authentication private key
export EIGENDA_AUTH_PK="your_private_key_here"

# Optional: NATS URL (defaults to localhost:4222)
export NATS_URL="nats://localhost:4222"
```

Generate your private key by running `generate_key.go`

### 2. Install Dependencies

```bash
go mod tidy
```

## Testing

### Manual Testing

To manually test the EigenDA integration:

```bash
# Set environment variables
export EIGENDA_AUTH_PK="your_private_key_here"
export RUN_EIGENDA_TEST=true

# Run the test
go test -v -run TestEigenDAIntegration ./da_layer
```

### What the Test Does

1. Creates a new DataAvailabilityService
2. Sets up NATS subscriptions to listen for data events
3. Stores test data on EigenDA
4. Waits for the "data stored" event
5. Retrieves the data from EigenDA
6. Verifies the retrieved data matches what was stored
7. Waits for the "data retrieved" event

### Expected Output

If everything is working correctly, you should see output similar to:

```
=== RUN   TestEigenDAIntegration
Setting up test environment for EigenDA tests
Storing data in EigenDA...
Current Blob Status: {
  "status":  "PROCESSING",
  "info":  {}
}
...
Current Blob Status: {
  "status":  "CONFIRMED",
  "info":  {
    "blobHeader": {...},
    "blobVerificationProof": {...}
  }
}
Data stored with ID: f9c979e84c19929dcdfc0c4f7ba65dc3ab47276e6d910480ed2d84ccbd4b8a3d...
Event received: Data stored with ID: f9c979e84c19929dcdfc0c4f7ba65dc3ab47276e6d910480ed2d84ccbd4b8a3d...
Received data stored event for ID: f9c979e84c19929dcdfc0c4f7ba65dc3ab47276e6d910480ed2d84ccbd4b8a3d...
Retrieving data from EigenDA...
Retrieved data: map[message:Hello EigenDA with NATS! testId:test-1234567890 timestamp:1234567890]
Event received: Data retrieved with ID: f9c979e84c19929dcdfc0c4f7ba65dc3ab47276e6d910480ed2d84ccbd4b8a3d...
Received data retrieved event
EigenDA integration test completed successfully!
--- PASS: TestEigenDAIntegration (120.45s)
PASS
```

## Troubleshooting

### Common Issues

1. **Authentication Error**: Make sure your `EIGENDA_AUTH_PK` is correctly set and valid.

2. **NATS Connection Error**: Ensure NATS server is running and accessible.

3. **Timeout Waiting for Blob Status**: EigenDA can sometimes take longer than expected to process blobs. Try increasing the `EIGENDA_MAX_WAIT_TIME` constant in the code.

4. **Retrieval Error**: If you can store but not retrieve data, check that the blob has been fully finalized on EigenDA before attempting retrieval.

### Logs

The service outputs detailed logs about blob status during the dispersal process. These can be helpful for diagnosing issues. 