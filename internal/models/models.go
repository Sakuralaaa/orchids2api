package models

import (
	"sync"
	"time"
)

// Model represents a supported AI model
type Model struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Provider    string    `json:"provider"`
	Description string    `json:"description"`
	MaxTokens   int       `json:"max_tokens"`
	Available   bool      `json:"available"`
	LastChecked time.Time `json:"last_checked,omitempty"`
}

// ModelRegistry manages available models
type ModelRegistry struct {
	models map[string]*Model
	mu     sync.RWMutex
}

// SupportedModels defines all models supported by the system
var SupportedModels = []*Model{
	{
		ID:          "gemini-3-flash",
		Name:        "Gemini 3 Flash",
		Provider:    "google",
		Description: "Google's fast and efficient Gemini 3 Flash model",
		MaxTokens:   128000,
		Available:   true,
	},
	{
		ID:          "claude-opus-4.5",
		Name:        "Claude Opus 4.5",
		Provider:    "anthropic",
		Description: "Anthropic's most capable Claude model",
		MaxTokens:   200000,
		Available:   true,
	},
	{
		ID:          "claude-sonnet-4.5",
		Name:        "Claude Sonnet 4.5",
		Provider:    "anthropic",
		Description: "Anthropic's balanced Claude model for most tasks",
		MaxTokens:   200000,
		Available:   true,
	},
	{
		ID:          "gpt-5.2-codex",
		Name:        "GPT-5.2 Codex",
		Provider:    "openai",
		Description: "OpenAI's advanced code generation model",
		MaxTokens:   128000,
		Available:   true,
	},
}

// ModelAliases maps common model names to internal model IDs
var ModelAliases = map[string]string{
	// Claude aliases
	"claude-3-opus-20240229":     "claude-opus-4.5",
	"claude-3-5-opus-20240620":   "claude-opus-4.5",
	"claude-opus":                "claude-opus-4.5",
	"opus":                       "claude-opus-4.5",
	"claude-3-sonnet-20240229":   "claude-sonnet-4.5",
	"claude-3-5-sonnet-20240620": "claude-sonnet-4.5",
	"claude-sonnet":              "claude-sonnet-4.5",
	"sonnet":                     "claude-sonnet-4.5",
	"claude-3-haiku-20240307":    "gemini-3-flash",
	"haiku":                      "gemini-3-flash",
	// GPT aliases
	"gpt-4":       "gpt-5.2-codex",
	"gpt-4-turbo": "gpt-5.2-codex",
	"gpt-4o":      "gpt-5.2-codex",
	"gpt-5":       "gpt-5.2-codex",
	"codex":       "gpt-5.2-codex",
	// Gemini aliases
	"gemini":       "gemini-3-flash",
	"gemini-pro":   "gemini-3-flash",
	"gemini-flash": "gemini-3-flash",
}

// NewRegistry creates a new model registry
func NewRegistry() *ModelRegistry {
	r := &ModelRegistry{
		models: make(map[string]*Model),
	}
	for _, m := range SupportedModels {
		r.models[m.ID] = m
	}
	return r
}

// GetModel returns a model by ID
func (r *ModelRegistry) GetModel(id string) (*Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check direct ID
	if model, ok := r.models[id]; ok {
		return model, true
	}

	// Check aliases
	if aliasedID, ok := ModelAliases[id]; ok {
		if model, ok := r.models[aliasedID]; ok {
			return model, true
		}
	}

	return nil, false
}

// ListModels returns all available models
func (r *ModelRegistry) ListModels() []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*Model, 0, len(r.models))
	for _, m := range r.models {
		models = append(models, m)
	}
	return models
}

// ListAvailableModels returns only available models
func (r *ModelRegistry) ListAvailableModels() []*Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*Model, 0)
	for _, m := range r.models {
		if m.Available {
			models = append(models, m)
		}
	}
	return models
}

// SetModelAvailability updates a model's availability status
func (r *ModelRegistry) SetModelAvailability(id string, available bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if model, ok := r.models[id]; ok {
		model.Available = available
		model.LastChecked = time.Now()
	}
}

// ResolveModel resolves a model name/alias to the actual model ID
func (r *ModelRegistry) ResolveModel(name string) string {
	// Check if it's a direct model ID
	if _, ok := r.models[name]; ok {
		return name
	}

	// Check aliases
	if aliasedID, ok := ModelAliases[name]; ok {
		return aliasedID
	}

	// Default fallback
	return "claude-sonnet-4.5"
}

// DefaultModel returns the default model
func (r *ModelRegistry) DefaultModel() *Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if model, ok := r.models["claude-sonnet-4.5"]; ok {
		return model
	}
	return SupportedModels[0]
}
