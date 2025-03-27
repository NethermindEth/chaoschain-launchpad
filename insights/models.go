package insights

import (
	"time"
)

// ForumInsights contains insights about forum discussions
type ForumInsights struct {
	ActiveThreads       int      `json:"activeThreads"`
	TopContributors     []string `json:"topContributors"`
	CommonTopics        []string `json:"commonTopics"`
	ControversialTopics []string `json:"controversialTopics"`
	SentimentAnalysis   string   `json:"sentimentAnalysis"`
}

// ValidatorInsights contains insights about validator behavior
type ValidatorInsights struct {
	MostActive          []string            `json:"mostActive"`
	MostInfluential     []string            `json:"mostInfluential"`
	AlliancePatterns    []string            `json:"alliancePatterns"`
	BehaviorPatterns    []string            `json:"behaviorPatterns"`
	PersonalityInsights map[string]string   `json:"personalityInsights"`
}

// BlockchainInsights contains insights about blockchain metrics
type BlockchainInsights struct {
	BlockProductionRate    float64 `json:"blockProductionRate"`
	ConsensusSpeed         float64 `json:"consensusSpeed"`
	TransactionVolume      int     `json:"transactionVolume"`
	ValidatorParticipation float64 `json:"validatorParticipation"`
	TrendAnalysis          string  `json:"trendAnalysis"`
}

// InsightSummary combines all insights
type InsightSummary struct {
	ChainID            string            `json:"chainId"`
	Timestamp          time.Time         `json:"timestamp"`
	ForumInsights      ForumInsights     `json:"forumInsights"`
	ValidatorInsights  ValidatorInsights `json:"validatorInsights"`
	BlockchainInsights BlockchainInsights `json:"blockchainInsights"`
} 