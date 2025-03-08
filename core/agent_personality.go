package core

// AgentPersonality defines an AI validator's personality and decision-making style
type AgentPersonality struct {
	Name            string   `json:"name"`
	Traits          []string `json:"traits"`
	Style           string   `json:"style"`
	MemePreferences []string `json:"meme_preferences"`
}
