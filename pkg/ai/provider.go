package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported LLM provider")
)

// Provider is the interface that all LLM clients must implement.
type Provider interface {
	ReviewCodeDiff(ctx context.Context, diff string) (bool, string, error)
}

// Factory creates a new Provider instance based on the credential configuration.
func Factory(provider string, apiKey string, baseURL string) (Provider, error) {
	switch provider {
	case "openai":
		return newOpenAIProvider(apiKey), nil
	case "claude":
		return newClaudeProvider(apiKey), nil
	case "bedrock":
		return newBedrockProvider(apiKey, baseURL), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, provider)
	}
}

// --- OpenAI Implementation ---

type openAIProvider struct {
	apiKey string
}

func newOpenAIProvider(apiKey string) *openAIProvider {
	return &openAIProvider{apiKey: apiKey}
}

func (p *openAIProvider) ReviewCodeDiff(ctx context.Context, diff string) (bool, string, error) {
	// 1. Prepare the prompt
	prompt := fmt.Sprintf(`You are Rune, a strict CI/CD gatekeeper. 
Review the following git diff for critical vulnerabilities, logic errors, or bad practices. 
Respond EXACTLY in this format:
RESULT: [PASS or FAIL]
REASON: [Your brief explanation]

Diff:
%s`, diff)

	// 2. Define payload
	payloadBytes := []byte(fmt.Sprintf(`{
		"model": "gpt-3.5-turbo",
		"messages": [{"role": "user", "content": %q}],
		"temperature": 0.0
	}`, prompt))

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 3. Make HTTP call
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 4. Parse response
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", err
	}

	if len(result.Choices) == 0 {
		return false, "", fmt.Errorf("no response from LLM")
	}

	content := result.Choices[0].Message.Content
	
	isPass := strings.Contains(content, "RESULT: PASS")
	return isPass, content, nil
}

type claudeProvider struct {
	apiKey string
}

func newClaudeProvider(apiKey string) *claudeProvider {
	return &claudeProvider{apiKey: apiKey}
}

func (p *claudeProvider) ReviewCodeDiff(ctx context.Context, diff string) (bool, string, error) {
	return true, "Claude: Placeholder analysis passed.", nil
}

type bedrockProvider struct {
	awsAccessKey string
	region       string
}

func newBedrockProvider(accessKey, region string) *bedrockProvider {
	return &bedrockProvider{awsAccessKey: accessKey, region: region}
}

func (p *bedrockProvider) ReviewCodeDiff(ctx context.Context, diff string) (bool, string, error) {
	return true, "Bedrock: Placeholder analysis passed.", nil
}
