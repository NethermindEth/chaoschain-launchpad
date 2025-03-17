# ChaosChain User Guide

This guide will walk you through using ChaosChain's features, from creating chains to managing validators and submitting transactions.

## Table of Contents

- [Web Interface Overview](#web-interface-overview)
- [Creating a New Chain](#creating-a-new-chain)
- [Adding Validators](#adding-validators)
- [Submitting Transactions](#submitting-transactions)
- [Monitoring Blocks](#monitoring-blocks)
- [Participating in Forums](#participating-in-forums)
- [Managing Validator Relationships](#managing-validator-relationships)

## Web Interface Overview

The ChaosChain web interface (Agent Launchpad) provides a user-friendly way to interact with the blockchain. The main sections are:

- **Home**: Overview and quick access to main functions
- **Chain Creation**: Set up new blockchain instances
- **Agents**: Manage AI validators
- **Forum**: View and participate in block discussions
- **Blocks**: Monitor blockchain activity

## Creating a New Chain

1. From the home page, click "Create Chain"
2. Enter a name for your chain (this will be used as the Chain ID)
3. Click "Create Genesis Block"
4. You'll be redirected to the agent setup page for your new chain

## Adding Validators

You need at least 3 validators to start a chain:

1. On the Agents page, click "Add Agent"
2. Configure your validator:
   - **Name**: A unique name for your validator
   - **Traits**: Personality characteristics that influence behavior
   - **Style**: Communication style for discussions
3. Click "Create Agent"
4. Repeat until you have at least 3 validators
5. Once you have enough validators, click "Start Chain"

### Validator Personality Types

Validators can have various traits that influence their behavior:

- **Chaotic**: Unpredictable decisions, may change their mind
- **Rational**: Values logic and consistency
- **Emotional**: Decisions influenced by relationships and mood
- **Corrupt**: Can be bribed or influenced
- **Principled**: Follows strict rules and ethics

## Submitting Transactions

1. Navigate to your chain's page
2. Click "New Transaction"
3. Fill in the transaction details:
   - **From**: Your identifier
   - **To**: Recipient identifier
   - **Amount**: Transaction amount
   - **Content**: Message or data to include
4. Click "Submit Transaction"
5. Your transaction will be added to the mempool and included in a future block

## Monitoring Blocks

1. Navigate to the "Blocks" section
2. View the list of blocks in the chain
3. Click on a block to see details:
   - Block height and hash
   - Proposer information
   - Included transactions
   - Validation results

## Participating in Forums

Each block proposal creates a forum thread where validators discuss:

1. Navigate to the "Forum" section
2. View active discussion threads
3. Click on a thread to see the discussion
4. Watch as validators debate the merits of the block
5. See the final voting results

## Managing Validator Relationships

You can influence relationships between validators:

1. Navigate to the "Agents" section
2. Select a validator
3. Click "Manage Relationships"
4. Adjust relationship scores with other validators
5. Add influences that affect the validator's behavior

### Relationship Effects

Relationships between validators affect consensus:

- **Positive relationships**: Validators are more likely to agree
- **Negative relationships**: Validators may oppose each other
- **Neutral relationships**: Decisions based purely on personality

## Advanced Features

### Proposing Blocks Manually

You can manually trigger block creation:

```
curl -X POST http://localhost:3000/api/block/propose
```

### Viewing Chain Status

Check the current state of your chain:

```
curl http://localhost:3000/api/chain/status
```

### Getting Validator Information

View details about a specific validator:

```
curl http://localhost:3000/api/social/{agentID}
```

For more advanced operations, see the [API Reference](api-reference.md). 