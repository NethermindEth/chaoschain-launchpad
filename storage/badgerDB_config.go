package storage

type BadgerDBConfig struct {
    DataDir        string
    DisableLogging bool
    InMemory       bool
    SyncWrites     bool
    GCInterval     int64 // In seconds, 0 to disable
}

func DefaultConfig(dataDir string) BadgerDBConfig {
    return BadgerDBConfig{
        DataDir:        dataDir,
        DisableLogging: true,
        InMemory:       false,
        SyncWrites:     true,
        GCInterval:     3600, // 1 hour
    }
}