package storage

import (
    "fmt"
	"encoding/json"
    
    "github.com/NethermindEth/chaoschain-launchpad/mempool"
)

type VoteRepository struct {
    db *DBStorage
}

func NewVoteRepository(db *DBStorage) *VoteRepository {
    return &VoteRepository{db: db}
}

func (r *VoteRepository) SaveVote(chainID string, vote mempool.EphemeralVote) error {
    key := fmt.Sprintf("vote:%s:%s", chainID, vote.ID)
    return r.db.PutObject(key, vote)
}

func (r *VoteRepository) GetVotes(chainID string) ([]mempool.EphemeralVote, error) {
    prefix := fmt.Sprintf("vote:%s:", chainID)
    data, err := r.db.GetByPrefix(prefix)
    if err != nil {
        return nil, err
    }
    
    votes := make([]mempool.EphemeralVote, 0, len(data))
    for _, v := range data {
        var vote mempool.EphemeralVote
        if err := json.Unmarshal(v, &vote); err != nil {
            continue // Skip invalid entries
        }
        votes = append(votes, vote)
    }
    return votes, nil
}

func (r *VoteRepository) DeleteVotes(chainID string) error {
    prefix := fmt.Sprintf("vote:%s:", chainID)
    return r.db.DeleteByPrefix(prefix)
}