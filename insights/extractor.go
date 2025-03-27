package insights

import (
	"fmt"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/consensus"
)

type DiscussionAnalysis struct {
	Analysis    string    `json:"analysis"`
	LastUpdated time.Time `json:"lastUpdated"`
}

type MessageJSON struct {
	Stance  string `json:"stance"`
	Reason  string `json:"reason"`
}

// Extractor analyzes blockchain discussions
type Extractor struct {
	apiBaseURL string
}

// NewExtractor creates a new insights extractor
func NewExtractor(apiBaseURL string) *Extractor {
	return &Extractor{
		apiBaseURL: apiBaseURL,
	}
}

// AnalyzeDiscussions analyzes all discussions across rounds
func (e *Extractor) AnalyzeDiscussions(chainID string) (*DiscussionAnalysis, error) {
	fmt.Printf("\n=== AnalyzeDiscussions Start ===\n")
	
	// Get discussions from consensus
	cm := consensus.GetConsensusManager(chainID)
	if cm == nil {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	activeConsensus := cm.GetActiveConsensus()
	if activeConsensus == nil {
		return nil, fmt.Errorf("no active consensus found")
	}

	discussions := activeConsensus.GetDiscussions()
	if len(discussions) == 0 {
		return nil, fmt.Errorf("no discussions found")
	}

	// Generate analysis
	analysis, err := generateDiscussionAnalysis(discussions)
	if err != nil {
		return nil, err
	}

	return &DiscussionAnalysis{
		Analysis:    analysis,
		LastUpdated: time.Now(),
	}, nil
} 