# Development Guide

This guide provides information for developers who want to contribute to ChaosChain or build applications on top of it.

## Development Environment Setup

### Prerequisites

- Go 1.24+
- Node.js 19+
- Docker
- Git
- OpenAI API key

### Setting Up the Development Environment

1. Clone the repository:

```
git clone https://github.com/NethermindEth/chaoschain-launchpad.git
cd chaoschain-launchpad
```

2. Install Go dependencies:

```
go mod download
```

3. Set up the frontend:

```
cd client/agent-launchpad
npm install
```

4. Create a `.env` file in the root directory:

```
OPENAI_API_KEY=your_api_key_here
```

5. Start NATS server:

```
docker run -p 4222:4222 -p 8222:8222 nats
```

## Project Structure

```
chaoschain-launchpad/
├── ai/                 # AI integration
├── api/                # API endpoints
│   └── handlers/       # Request handlers
├── client/             # Web interface
│   └── agent-launchpad/# Next.js application
├── cmd/                # Command-line tools
│   ├── keygen/         # Key generation utility
│   └── main.go         # Main application entry point
├── communication/      # Messaging components
├── config/             # Configuration management
├── consensus/          # Consensus implementation
├── core/               # Core blockchain components
├── crypto/             # Cryptographic utilities
├── docs/               # Documentation
├── mempool/            # Transaction pool
├── p2p/                # Peer-to-peer networking
├── producer/           # Block producer
├── validator/          # Validator implementation
├── go.mod              # Go module definition
├── go.sum              # Go dependencies checksum
├── main.go             # Alternative entry point
└── README.md           # Project overview
```

## Adding a New Feature

### Backend (Go)

1. Identify which package should contain your feature
2. Create or modify the necessary files
3. Add tests in a `_test.go` file
4. Update any relevant documentation
5. Run tests:

```
go test ./...
```

### Frontend (Next.js)

1. Navigate to the client directory:

```
cd client/agent-launchpad
```

2. Create or modify components in `src/components/`
3. Add new pages in `src/app/`
4. Update API services in `src/services/`
5. Run the development server:

```
npm run dev
```

## Extending ChaosChain

### Creating Custom Validator Personalities

You can extend the validator system by adding new personality traits:

1. Modify `validator/validator.go` to include new traits
2. Update the AI prompts in `ai/ai.go` to incorporate these traits
3. Add UI elements to the web interface for configuring these traits

### Adding New Transaction Types

To add custom transaction types:

1. Extend the `Transaction` struct in `core/transaction.go`
2. Add validation logic for the new transaction type
3. Update the mempool to handle the new transaction type
4. Add API endpoints for submitting the new transaction type

### Implementing Custom Consensus Rules

To modify the consensus mechanism:

1. Update the discussion logic in `consensus/discussion.go`
2. Modify the voting process in `consensus/manager.go`
3. Adjust the finalization criteria as needed

## Testing

### Running Tests

```
# Run all tests
go test ./...

# Run tests for a specific package
go test ./core

# Run tests with coverage
go test -cover ./...
```

### Testing the Web Interface

```
cd client/agent-launchpad
npm test
```

## Building for Production

### Backend

```
go build -o chaoschain cmd/main.go
```

### Frontend

```
cd client/agent-launchpad
npm run build
```

## Deployment

See the [Deployment Guide](deployment.md) for information on deploying ChaosChain to production environments.

## Contributing

1. Fork the repository
2. Create a feature branch:

```
git checkout -b feature/my-new-feature
```

3. Make your changes
4. Run tests
5. Commit your changes:

```
git commit -am 'Add some feature'
```

6. Push to the branch:

```
git push origin feature/my-new-feature
```

7. Create a new Pull Request

Please follow the [Contribution Guidelines](../CONTRIBUTING.md) for more details. 