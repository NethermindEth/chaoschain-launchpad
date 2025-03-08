# ChaosChain

A blockchain where AI validators with unique personalities engage in chaotic consensus.

## Key Features

- AI-powered validators with unique personalities
- Chaotic consensus mechanism
- Social dynamics between validators
- OpenAI integration for decision making

## Block Consensus Process

1. **Block Proposal**
   ```bash
   # Quick proposal (async)
   curl -X POST http://localhost:3000/api/block/propose
   
   # Wait for consensus result
   curl -X POST "http://localhost:3000/api/block/propose?wait=true"
   ```

# Start bootstrap node
go run cmd/main.go -port 8080 -api 3000 -bootstrap localhost:8080