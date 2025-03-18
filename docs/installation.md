# Installation Guide

This guide will walk you through the process of setting up ChaosChain on your local machine.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go** (version 1.24 or higher)
- **Node.js** (version 19 or higher)
- **npm** or **yarn**
- **Docker** (optional, for running NATS server)
- **OpenAI API key**

## Step 1: Clone the Repository

```
git clone https://github.com/NethermindEth/chaoschain-launchpad.git
cd chaoschain-launchpad
```

## Step 2: Set Up Environment Variables

Create a `.env` file in the root directory with your OpenAI API key:

```
OPENAI_API_KEY=your_api_key_here
```

This key is required for the AI-powered validators to function.

## Step 3: Start NATS Server

ChaosChain uses NATS for messaging between components. You can run it using Docker:

```
docker run -p 4222:4222 -p 8222:8222 nats
```

Alternatively, you can [install NATS server directly](https://docs.nats.io/running-a-nats-service/introduction/installation).

## Step 4: Build and Run the Backend

```
# Build the project
go build -o chaoschain cmd/main.go

# Run the bootstrap node
./chaoschain -port 8080 -api 3000 -nats nats://localhost:4222
```

This starts the bootstrap node with:
- P2P network on port 8080
- API server on port 3000
- Connection to NATS server

## Step 5: Set Up the Web Interface

```
# Navigate to the client directory
cd client/agent-launchpad

# Install dependencies
npm install

# Start the development server
npm run dev
```

The web interface will be available at [http://localhost:4000](http://localhost:4000).

## Step 6: Create Your First Chain

1. Open the web interface at [http://localhost:4000](http://localhost:4000)
2. Click "Create Chain"
3. Enter a name for your chain
4. Add at least 3 AI validators
5. Start the chain

For more detailed instructions on using the web interface, see the [User Guide](user-guide.md).

## Running Multiple Nodes

To run additional nodes that connect to your bootstrap node:

```
./chaoschain -port 8081 -api 3001 -nats nats://localhost:4222 -bootstrap localhost:8080
```

## Troubleshooting

### Common Issues

**Error: OpenAI API key not found**
- Ensure your `.env` file contains the correct API key
- Check that the `.env` file is in the root directory

**Error: Failed to connect to NATS server**
- Verify that the NATS server is running
- Check the NATS URL in your command line arguments

**Error: Port already in use**
- Change the port numbers using the `-port` and `-api` flags

For more help, see the [Troubleshooting Guide](troubleshooting.md) or open an issue on GitHub. 