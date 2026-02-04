package models

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	// Check that all supported models are registered
	for _, m := range SupportedModels {
		if _, found := r.GetModel(m.ID); !found {
			t.Errorf("Expected model %s to be registered", m.ID)
		}
	}
}

func TestResolveModel(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		input    string
		expected string
	}{
		// Direct model IDs
		{"gemini-3-flash", "gemini-3-flash"},
		{"claude-opus-4.5", "claude-opus-4.5"},
		{"claude-sonnet-4.5", "claude-sonnet-4.5"},
		{"gpt-5.2-codex", "gpt-5.2-codex"},
		// Aliases
		{"opus", "claude-opus-4.5"},
		{"sonnet", "claude-sonnet-4.5"},
		{"haiku", "gemini-3-flash"},
		{"claude-opus", "claude-opus-4.5"},
		{"claude-sonnet", "claude-sonnet-4.5"},
		{"gpt-4", "gpt-5.2-codex"},
		{"gpt-5", "gpt-5.2-codex"},
		{"gemini", "gemini-3-flash"},
		// Unknown defaults to claude-sonnet-4.5
		{"unknown-model", "claude-sonnet-4.5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := r.ResolveModel(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveModel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestListAvailableModels(t *testing.T) {
	r := NewRegistry()

	models := r.ListAvailableModels()
	if len(models) != len(SupportedModels) {
		t.Errorf("Expected %d models, got %d", len(SupportedModels), len(models))
	}

	// Disable one model and check again
	r.SetModelAvailability("gemini-3-flash", false)
	models = r.ListAvailableModels()
	if len(models) != len(SupportedModels)-1 {
		t.Errorf("Expected %d available models after disabling one, got %d", len(SupportedModels)-1, len(models))
	}
}

func TestSetModelAvailability(t *testing.T) {
	r := NewRegistry()

	// Initially available
	model, found := r.GetModel("claude-opus-4.5")
	if !found {
		t.Fatal("Model claude-opus-4.5 not found")
	}
	if !model.Available {
		t.Error("Expected model to be available initially")
	}

	// Set to unavailable
	r.SetModelAvailability("claude-opus-4.5", false)
	model, _ = r.GetModel("claude-opus-4.5")
	if model.Available {
		t.Error("Expected model to be unavailable after setting")
	}

	// Set back to available
	r.SetModelAvailability("claude-opus-4.5", true)
	model, _ = r.GetModel("claude-opus-4.5")
	if !model.Available {
		t.Error("Expected model to be available after re-enabling")
	}
}

func TestGetModelWithAliases(t *testing.T) {
	r := NewRegistry()

	// Test getting model by alias
	model, found := r.GetModel("opus")
	if !found {
		t.Fatal("Model not found by alias 'opus'")
	}
	if model.ID != "claude-opus-4.5" {
		t.Errorf("Expected model ID 'claude-opus-4.5', got '%s'", model.ID)
	}
}

func TestDefaultModel(t *testing.T) {
	r := NewRegistry()

	defaultModel := r.DefaultModel()
	if defaultModel == nil {
		t.Fatal("DefaultModel returned nil")
	}
	if defaultModel.ID != "claude-sonnet-4.5" {
		t.Errorf("Expected default model 'claude-sonnet-4.5', got '%s'", defaultModel.ID)
	}
}
