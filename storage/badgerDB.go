package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic" 
	"time" 

	"github.com/NethermindEth/chaoschain-launchpad/core"
	"github.com/NethermindEth/chaoschain-launchpad/mempool"
	"github.com/dgraph-io/badger/v3"
)


type Storage interface {
    // Generic operations
    Put(key string, value []byte) error
    Get(key string) ([]byte, error)
    Delete(key string) error
    GetByPrefix(prefix string) (map[string][]byte, error)
    DeleteByPrefix(prefix string) error
    PutObject(key string, obj interface{}) error
    GetObject(key string, obj interface{}) error
    
    // Domain-specific operations
    SaveTransaction(chainID string, tx core.Transaction) error
    GetTransaction(chainID, txID string) (core.Transaction, error)
    DeleteTransaction(chainID, txID string) error
    SaveEphemeralVote(chainID string, vote mempool.EphemeralVote) error
    GetEphemeralVotes(chainID string) ([]mempool.EphemeralVote, error)
    SaveEphemeralBlockHash(chainID, blockHash string) error
    GetEphemeralBlockHashes(chainID string) ([]string, error)
    SaveAgentIdentity(chainID, agentID, identity string) error
    GetAgentIdentities(chainID string) (map[string]string, error)
    ClearChainData(chainID string) error
    
    // Management operations
    Close() error
    RunGC() error
}

type DBMetrics struct {
    PutCount        int64
    GetCount        int64
    DeleteCount     int64
    GetByPrefixCount int64
    Errors          int64
}

func (s *DBStorage) recordMetric(name string) {
    // Implementation depends on your metrics library
    // Example with atomic counters:
    switch name {
    case "put":
        atomic.AddInt64(&s.metrics.PutCount, 1)
    case "get":
        atomic.AddInt64(&s.metrics.GetCount, 1)
    // etc.
    }
}

func (s *DBStorage) logOperation(op string, key string, err error) {
    if err != nil {
        log.Printf("BadgerDB %s operation failed for key %s: %v", op, key, err)
        atomic.AddInt64(&s.metrics.Errors, 1)
    }
}

// DBStorage represents a persistent storage using BadgerDB
type DBStorage struct {
    db      *badger.DB
    mu      sync.Mutex
    config  BadgerDBConfig
    metrics DBMetrics
}

var (
	// Map of chainID -> DBStorage
	instances = make(map[string]*DBStorage)
	mu        sync.RWMutex
)

// GetDBStorage returns a DB instance for the specified chain
func GetDBStorage(dataDir, chainID string) (*DBStorage, error) {
    return GetDBStorageWithConfig(DefaultConfig(dataDir), chainID)
}

// GetDBStorageWithConfig returns a DB instance with custom configuration
func GetDBStorageWithConfig(config BadgerDBConfig, chainID string) (*DBStorage, error) {
    mu.RLock()
    instance, exists := instances[chainID]
    mu.RUnlock()

    if exists {
        return instance, nil
    }

    mu.Lock()
    defer mu.Unlock()

    // Check again in case another goroutine created it while we were waiting
    instance, exists = instances[chainID]
    if exists {
        return instance, nil
    }

    // Create a new instance
    dbPath := filepath.Join(config.DataDir, "badgerdb", chainID)
    instance, err := newDBStorage(dbPath, config)
    if err != nil {
        return nil, err
    }

    instances[chainID] = instance
    
    // Start GC if enabled
    if config.GCInterval > 0 {
        go instance.startGCRoutine(time.Duration(config.GCInterval) * time.Second)
    }
    
    return instance, nil
}

// newDBStorage creates a new BadgerDB storage instance
func newDBStorage(dbPath string, config BadgerDBConfig) (*DBStorage, error) {
    opts := badger.DefaultOptions(dbPath)
    if config.DisableLogging {
        opts.Logger = nil
    }
    opts.InMemory = config.InMemory
    opts.SyncWrites = config.SyncWrites

    db, err := badger.Open(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to open BadgerDB: %v", err)
    }

    return &DBStorage{
        db:     db,
        config: config,
    }, nil
}

func (s *DBStorage) startGCRoutine(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for range ticker.C {
        err := s.RunGC()
        if err != nil {
            log.Printf("BadgerDB GC failed: %v", err)
        }
    }
}

// Close closes the BadgerDB database
func (s *DBStorage) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// CloseAll closes all BadgerDB instances
func CloseAll() {
	mu.Lock()
	defer mu.Unlock()

	for _, instance := range instances {
		instance.Close()
	}
	instances = make(map[string]*DBStorage)
}

// Put stores a key-value pair in the database
func (s *DBStorage) Put(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
}

// Get retrieves a value from the database by key
func (s *DBStorage) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil // Key not found, return nil value
			}
			return err
		}

		return item.Value(func(val []byte) error {
			valCopy = append([]byte{}, val...)
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get value: %v", err)
	}

	return valCopy, nil
}

// Delete removes a key-value pair from the database
func (s *DBStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// GetByPrefix retrieves all key-value pairs with a given prefix
func (s *DBStorage) GetByPrefix(prefix string) (map[string][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string][]byte)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				// Copy the key and value since they are only valid during this transaction
				keyCopy := append([]byte{}, k...)
				valCopy := append([]byte{}, v...)
				result[string(keyCopy)] = valCopy
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get values by prefix: %v", err)
	}

	return result, nil
}

// DeleteByPrefix deletes all key-value pairs with a given prefix
func (s *DBStorage) DeleteByPrefix(prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.deleteByPrefix(prefix)
}

// PutObject serializes and stores an object in the database
func (s *DBStorage) PutObject(key string, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal object: %v", err)
	}

	return s.Put(key, data)
}

// GetObject retrieves and deserializes an object from the database
func (s *DBStorage) GetObject(key string, obj interface{}) error {
	data, err := s.Get(key)
	if err != nil {
		return err
	}

	if data == nil {
		return fmt.Errorf("key not found: %s", key)
	}

	if err := json.Unmarshal(data, obj); err != nil {
		return fmt.Errorf("failed to unmarshal object: %v", err)
	}

	return nil
}

// RunGC runs garbage collection on the database
func (s *DBStorage) RunGC() error {
	return s.db.RunValueLogGC(0.5) // Clean up if at least 50% can be discarded
}

// SaveTransaction persists a transaction to BadgerDB
func (s *DBStorage) SaveTransaction(chainID string, tx core.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("tx:%s:%s", chainID, tx.Signature)
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetTransaction retrieves a transaction from BadgerDB
func (s *DBStorage) GetTransaction(chainID, txID string) (core.Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var tx core.Transaction
	key := fmt.Sprintf("tx:%s:%s", chainID, txID)

	var data []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("transaction not found")
			}
			return err
		}

		var valErr error
		data, valErr = item.ValueCopy(nil)
		return valErr
	})
	if err != nil {
		return tx, err
	}

	if err := json.Unmarshal(data, &tx); err != nil {
		return tx, fmt.Errorf("failed to unmarshal transaction: %v", err)
	}

	return tx, nil
}

// DeleteTransaction removes a transaction from BadgerDB
func (s *DBStorage) DeleteTransaction(chainID, txID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("tx:%s:%s", chainID, txID)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// SaveEphemeralVote persists an ephemeral vote to BadgerDB
func (s *DBStorage) SaveEphemeralVote(chainID string, vote mempool.EphemeralVote) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("vote:%s:%s", chainID, vote.ID)
	data, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("failed to marshal vote: %v", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

// GetEphemeralVotes retrieves all ephemeral votes for a chain from BadgerDB
func (s *DBStorage) GetEphemeralVotes(chainID string) ([]mempool.EphemeralVote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := fmt.Sprintf("vote:%s:", chainID)
	var votes []mempool.EphemeralVote
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var vote mempool.EphemeralVote
				if err := json.Unmarshal(v, &vote); err != nil {
					log.Printf("Failed to unmarshal vote: %v", err)
					return nil
				}
				votes = append(votes, vote)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get ephemeral votes: %v", err)
	}

	return votes, nil
}

// SaveEphemeralBlockHash persists a block hash to BadgerDB
func (s *DBStorage) SaveEphemeralBlockHash(chainID, blockHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("blockhash:%s:%s", chainID, blockHash)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(blockHash))
	})
}

// GetEphemeralBlockHashes retrieves all ephemeral block hashes for a chain from BadgerDB
func (s *DBStorage) GetEphemeralBlockHashes(chainID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := fmt.Sprintf("blockhash:%s:", chainID)
	var hashes []string
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				hashes = append(hashes, string(v))
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get ephemeral block hashes: %v", err)
	}

	return hashes, nil
}

// SaveAgentIdentity persists an agent identity to BadgerDB
func (s *DBStorage) SaveAgentIdentity(chainID, agentID, identity string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("agent:%s:%s", chainID, agentID)
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(identity))
	})
}

// GetAgentIdentities retrieves all agent identities for a chain from BadgerDB
func (s *DBStorage) GetAgentIdentities(chainID string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := fmt.Sprintf("agent:%s:", chainID)
	identities := make(map[string]string)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				agentID := string(k[len(prefix):])
				identities[agentID] = string(v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get agent identities: %v", err)
	}

	return identities, nil
}

// ClearChainData removes all data for a specific chain
func (s *DBStorage) ClearChainData(chainID string) error {
	// This is a simplified implementation - in production, you might want to use batches
	prefixes := []string{
		fmt.Sprintf("tx:%s:", chainID),
		fmt.Sprintf("vote:%s:", chainID),
		fmt.Sprintf("blockhash:%s:", chainID),
		fmt.Sprintf("agent:%s:", chainID),
	}

	for _, prefix := range prefixes {
		if err := s.deleteByPrefix(prefix); err != nil {
			return err
		}
	}

	return nil
}

// deleteByPrefix deletes all keys with the given prefix
func (s *DBStorage) deleteByPrefix(prefix string) error {
	// First collect all keys to delete
	keysToDelete := [][]byte{}

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(prefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysToDelete = append(keysToDelete, key)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to collect keys for deletion: %v", err)
	}

	// Now delete all collected keys in a separate transaction
	return s.db.Update(func(txn *badger.Txn) error {
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return fmt.Errorf("failed to delete key: %v", err)
			}
		}
		return nil
	})
}
