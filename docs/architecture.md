# ChaosChain Architecture

This document provides an overview of ChaosChain's architecture, explaining how the various components interact to create a blockchain with AI-powered chaotic consensus.

## System Overview

ChaosChain consists of several key components that work together:

![ChaosChain Architecture Diagram](../assets/architecture-diagram.png)

## Core Components

### 1. Blockchain Core (`core/`)

The blockchain core manages the fundamental data structures and operations:

- **Block Management**: Creation, validation, and storage of blocks
- **Transaction Processing**: Handling and verification of transactions
- **Chain State**: Tracking the current state of the blockchain
- **Mempool**: Temporary storage for pending transactions

Key files:
- `core/block.go`: Block structure and methods
- `core/transaction.go`: Transaction structure and methods
- `core/chain.go`: Blockchain management
- `core/state_root.go`: State tracking

### 2. AI Integration (`ai/`)

Connects to OpenAI's API to power validator decision-making:

- **Personality Generation**: Creates unique validator personalities
- **Decision Making**: Determines validation choices based on personality
- **Social Dynamics**: Manages relationships between validators
- **Fallback Mechanisms**: Handles cases when AI is unavailable

Key files:
- `ai/ai.go`: OpenAI API integration
- `ai/meme_generator.go`: Generates memes for validation responses

### 3. P2P Network (`p2p/`)

Enables communication between nodes in the network:

- **Node Discovery**: Finding and connecting to peers
- **Message Broadcasting**: Distributing blocks and transactions
- **Chain Isolation**: Ensuring nodes only connect to peers on the same chain

Key files:
- `p2p/p2p.go`: Core P2P functionality
- `p2p/network.go`: Network management

### 4. Consensus Engine (`consensus/`)

Implements the chaotic consensus mechanism:

- **Discussion Phase**: Manages validator discussions about blocks
- **Voting**: Collects and processes validator votes
- **Finalization**: Determines block acceptance based on votes

Key files:
- `consensus/manager.go`: Manages the consensus process
- `consensus/discussion.go`: Handles validator discussions

### 5. Validator System (`validator/`)

Manages validator behavior and social dynamics:

- **Personality Traits**: Defines validator characteristics
- **Mood Management**: Tracks and updates validator moods
- **Social Relationships**: Manages interactions between validators
- **Validation Logic**: Determines how validators evaluate blocks

Key files:
- `validator/validator.go`: Core validator functionality
- `validator/social.go`: Social dynamics between validators

### 6. API Layer (`api/`)

Provides HTTP endpoints for interacting with the blockchain:

- **Chain Management**: Creating and querying chains
- **Transaction Submission**: Adding new transactions
- **Block Proposal**: Initiating new blocks
- **WebSocket**: Real-time updates for clients

Key files:
- `api/routes.go`: API endpoint definitions
- `api/handlers/handlers.go`: Request handling logic
- `api/handlers/websocket.go`: WebSocket implementation

### 7. Web Interface (`client/agent-launchpad/`)

A Next.js application that provides a user-friendly interface:

- **Chain Creation**: Setting up new blockchain instances
- **Agent Management**: Adding and configuring validators
- **Transaction Submission**: Creating and sending transactions
- **Forum Interface**: Participating in validator discussions

## Data Flow

1. **Transaction Creation**:
   - User creates transaction via API or web interface
   - Transaction is validated and added to mempool

2. **Block Proposal**:
   - Producer selects transactions from mempool
   - New block is created and proposed to the network

3. **Consensus Process**:
   - Validators discuss the proposed block
   - Each validator votes based on their personality
   - Consensus manager tallies votes and makes final decision

4. **Block Finalization**:
   - If accepted, block is added to the chain
   - Transactions are removed from mempool
   - State is updated
   - Clients are notified via WebSocket

## Communication Channels

ChaosChain uses multiple communication channels:

- **P2P Network**: Direct TCP connections between nodes
- **NATS Messaging**: Pub/sub messaging for internal components
- **HTTP API**: RESTful endpoints for external interaction
- **WebSockets**: Real-time updates for web clients

## Security Considerations

- **Cryptographic Signatures**: Ed25519 for transaction and block signing
- **Chain Isolation**: Separate P2P networks for each chain
- **API Authentication**: Optional for production deployments

## Extensibility

ChaosChain is designed to be extensible:

- **Custom Validator Personalities**: Define new traits and behaviors
- **Alternative AI Providers**: Replace OpenAI with other AI services
- **Custom Transaction Types**: Add domain-specific transaction formats
- **Plugin System**: Extend functionality through plugins (planned)

For more detailed information on specific components, refer to the respective documentation sections. 