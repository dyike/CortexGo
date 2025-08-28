package utils

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed prompts
var promptFiles embed.FS

// LoadPrompt loads a prompt from the embedded markdown files
func LoadPrompt(path string) (string, error) {
	content, err := promptFiles.ReadFile(fmt.Sprintf("prompts/%s.md", path))
	if err != nil {
		return "", fmt.Errorf("failed to load prompt %s: %w", path, err)
	}
	return string(content), nil
}

// LoadPromptWithContext loads a prompt and replaces context variables
func LoadPromptWithContext(path string, context map[string]string) (string, error) {
	content, err := LoadPrompt(path)
	if err != nil {
		return "", err
	}

	// Replace context variables in the format {{.VariableName}}
	for key, value := range context {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content, nil
}
