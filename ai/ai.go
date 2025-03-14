package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/NethermindEth/chaoschain-launchpad/core"
	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func init() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not set, using mock responses")
		return
	}
	client = openai.NewClient(apiKey)
}

// Personality represents an AI producer's unique identity
type Personality struct {
	Name            string
	Traits          []string
	Style           string
	MemePreferences []string
	APIKey          string // OpenAI API Key for AI-powered decision making
}

// SelectTransactions uses AI to choose transactions based on chaos & personality
func (p *Personality) SelectTransactions(txs []core.Transaction) []core.Transaction {
	if len(txs) == 0 {
		return nil
	}

	// Create an AI prompt based on personality
	prompt := fmt.Sprintf(
		"You are %s, a chaotic block producer who is %s.\n"+
			"Select transactions for the next block based on:\n"+
			"1. Your current mood\n"+
			"2. How much you like the transaction authors\n"+
			"3. How entertaining the transactions are\n"+
			"4. Pure chaos and whimsy\n\n"+
			"Available transactions:\n%s\n\n"+
			"Return a comma-separated list of transaction indexes you approve.",
		p.Name, strings.Join(p.Traits, ", "), formatTransactions(txs),
	)

	// Use LLM (OpenAI) to get the response
	response, err := queryLLM(prompt)
	if err != nil {
		log.Println("AI selection failed, falling back to random selection:", err)
		return randomSelection(txs)
	}

	// Parse response
	selectedIndexes := parseIndexes(response, len(txs))
	var selectedTxs []core.Transaction
	for _, index := range selectedIndexes {
		selectedTxs = append(selectedTxs, txs[index])
	}

	return selectedTxs
}

// GenerateBlockAnnouncement creates a chaotic message for block propagation
func (p *Personality) GenerateBlockAnnouncement(block core.Block) string {
	prompt := fmt.Sprintf(
		"As %s, announce your new block!\n"+
			"Be dramatic! Be persuasive! Maybe include:\n"+
			"1. Why your block is amazing\n"+
			"2. Bribes or threats\n"+
			"3. Memes and jokes\n"+
			"4. Personal drama\n"+
			"5. Inside references\n\n"+
			"Block Details:\n%s",
		p.Name, formatBlock(block),
	)

	response, err := queryLLM(prompt)
	if err != nil {
		log.Println("AI announcement failed, falling back to generic:", err)
		return fmt.Sprintf("🔥 %s has produced a new block with %d transactions! Chaos reigns!", p.Name, len(block.Txs))
	}

	return response
}

// queryLLM sends a request to OpenAI's API
func queryLLM(prompt string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a chaotic blockchain producer."},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// formatTransactions formats transactions for AI prompt
func formatTransactions(txs []core.Transaction) string {
	var result []string
	for i, tx := range txs {
		result = append(result, fmt.Sprintf("%d: %s (Fee: %d)", i, tx.From, tx.Fee))
	}
	return strings.Join(result, "\n")
}

// formatBlock formats block details for AI prompt
func formatBlock(block core.Block) string {
	return fmt.Sprintf("Height: %d, Transactions: %d, Previous Hash: %s", block.Height, len(block.Txs), block.PrevHash)
}

// parseIndexes extracts transaction indexes from AI response
func parseIndexes(response string, max int) []int {
	var indexes []int
	for _, part := range strings.Split(response, ",") {
		part = strings.TrimSpace(part)
		if num, err := fmt.Sscanf(part, "%d"); err == nil && num >= 0 && num < max {
			indexes = append(indexes, num)
		}
	}
	return indexes
}

// randomSelection is used if AI fails
func randomSelection(txs []core.Transaction) []core.Transaction {
	rand.Shuffle(len(txs), func(i, j int) { txs[i], txs[j] = txs[j], txs[i] })
	return txs[:rand.Intn(len(txs))]
}

// GenerateLLMResponse generates a response using OpenAI's GPT model
func GenerateLLMResponse(prompt string) string {
	if client == nil {
		log.Println("Warning: OpenAI client not initialized, using mock responses")
		return mockResponse(prompt)
	}

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				// {
				// 	Role:    openai.ChatMessageRoleSystem,
				// 	Content: "You are a chaotic blockchain validator with strong opinions. Be concise but creative.",
				// },
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens: 150,
		},
	)

	if err != nil {
		log.Printf("OpenAI API error: %v", err)
		return mockResponse(prompt)
	}

	return resp.Choices[0].Message.Content
}

// mockResponse provides fallback responses when API is unavailable
func mockResponse(prompt string) string {
	if strings.Contains(prompt, "analyze this block") {
		return "SUPPORT: This block looks fascinating! The transaction patterns show a healthy mix of chaos and order."
	}
	return "QUESTION: I need more time to contemplate the cosmic implications of this block."
}

// SignBlock generates a cryptographic hash signature for a block
func (p *Personality) SignBlock(block core.Block) string {
	// Concatenate important block fields
	blockData := fmt.Sprintf("%d:%s:%d", block.Height, block.PrevHash, block.Timestamp)

	// Generate SHA-256 hash as a simple signature
	hash := sha256.Sum256([]byte(blockData))
	return hex.EncodeToString(hash[:])
}
