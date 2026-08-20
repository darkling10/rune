package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/deployerai/deployer/internal/models"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported LLM provider")
)

// Provider is the interface that all LLM clients must implement.
// This allows the core logic to be agnostic of the specific LLM used.
type Provider interface {
	// AnalyzeVulnerability analyzes a security vulnerability and suggests a fix.
	AnalyzeVulnerability(ctx context.Context, report string) (string, error)

	// AssessReleaseRisk evaluates whether a minor version bump is safe to auto-deploy.
	AssessReleaseRisk(ctx context.Context, releaseNotes, testResults string) (bool, string, error)
}

// Factory creates a new Provider instance based on the credential configuration.
func Factory(cred *models.LLMCredential) (Provider, error) {
	switch cred.Provider {
	case models.ProviderOpenAI:
		return newOpenAIProvider(cred.APIKey), nil
	case models.ProviderClaude:
		return newClaudeProvider(cred.APIKey), nil
	case models.ProviderBedrock:
		return newBedrockProvider(cred.APIKey, cred.BaseURL), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, cred.Provider)
	}
}

// --- Dummy Implementations (To be expanded in later phases) ---

type openAIProvider struct {
	apiKey string
}

func newOpenAIProvider(apiKey string) *openAIProvider {
	return &openAIProvider{apiKey: apiKey}
}

func (p *openAIProvider) AnalyzeVulnerability(ctx context.Context, report string) (string, error) {
	return "OpenAI analysis: Upgrade dependency to v1.2.3", nil
}

func (p *openAIProvider) AssessReleaseRisk(ctx context.Context, releaseNotes, testResults string) (bool, string, error) {
	return true, "OpenAI assessment: Safe to deploy", nil
}

type claudeProvider struct {
	apiKey string
}

func newClaudeProvider(apiKey string) *claudeProvider {
	return &claudeProvider{apiKey: apiKey}
}

func (p *claudeProvider) AnalyzeVulnerability(ctx context.Context, report string) (string, error) {
	return "Claude analysis: Critical vulnerability, apply patch immediately", nil
}

func (p *claudeProvider) AssessReleaseRisk(ctx context.Context, releaseNotes, testResults string) (bool, string, error) {
	return true, "Claude assessment: Safe to deploy", nil
}

type bedrockProvider struct {
	awsAccessKey string
	region       string
}

func newBedrockProvider(accessKey, region string) *bedrockProvider {
	return &bedrockProvider{awsAccessKey: accessKey, region: region}
}

func (p *bedrockProvider) AnalyzeVulnerability(ctx context.Context, report string) (string, error) {
	return "Bedrock analysis: Apply mitigation step X", nil
}

func (p *bedrockProvider) AssessReleaseRisk(ctx context.Context, releaseNotes, testResults string) (bool, string, error) {
	return false, "Bedrock assessment: Manual review required", nil
}
