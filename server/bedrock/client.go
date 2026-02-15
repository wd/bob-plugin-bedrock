package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// Client wraps AWS Bedrock runtime client
type Client struct {
	runtime *bedrockruntime.Client
}

// TranslateRequest represents a translation request
type TranslateRequest struct {
	Text   string `json:"text"`
	From   string `json:"from"`
	To     string `json:"to"`
	Stream bool   `json:"stream"`
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content string `json:"content,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewClient creates a new Bedrock client
func NewClient(ctx context.Context, region, profile string) (*Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{
		runtime: bedrockruntime.NewFromConfig(cfg),
	}, nil
}

// defaultPromptTemplate is the default translation prompt
const defaultPromptTemplate = `Translate the following text from {from} to {to}. Output ONLY the translated text without any explanation or notes.

Text to translate:
{text}`

// buildPrompt creates the translation prompt
func buildPrompt(text, from, to, customPrompt string) string {
	fromLang := languageName(from)
	toLang := languageName(to)

	template := customPrompt
	if template == "" {
		template = defaultPromptTemplate
	}

	// Replace variables in the template
	result := strings.Replace(template, "{from}", fromLang, -1)
	result = strings.Replace(result, "{to}", toLang, -1)
	result = strings.Replace(result, "{text}", text, -1)

	return result
}

// languageName converts language code to full name
func languageName(code string) string {
	names := map[string]string{
		"en":      "English",
		"zh":      "Chinese",
		"zh-Hans": "Simplified Chinese",
		"zh-Hant": "Traditional Chinese",
		"ja":      "Japanese",
		"ko":      "Korean",
		"fr":      "French",
		"de":      "German",
		"es":      "Spanish",
		"it":      "Italian",
		"pt":      "Portuguese",
		"ru":      "Russian",
		"ar":      "Arabic",
		"th":      "Thai",
		"vi":      "Vietnamese",
		"auto":    "detected language",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

// getDefaultModel returns default model if none specified
func getDefaultModel(model string) string {
	if model == "" {
		return "us.anthropic.claude-3-5-haiku-20241022-v1:0"
	}
	return model
}

// Translate performs non-streaming translation
func (c *Client) Translate(ctx context.Context, req *TranslateRequest) (string, error) {
	prompt := buildPrompt(req.Text, req.From, req.To, req.Prompt)
	model := getDefaultModel(req.Model)

	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(model),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: prompt},
				},
			},
		},
	}

	output, err := c.runtime.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("bedrock converse failed: %w", err)
	}

	// Extract text from response
	if output.Output == nil {
		return "", fmt.Errorf("empty response from bedrock")
	}

	msgOutput, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return "", fmt.Errorf("unexpected response type from bedrock")
	}

	var result strings.Builder
	for _, block := range msgOutput.Value.Content {
		if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
			result.WriteString(textBlock.Value)
		}
	}

	return result.String(), nil
}

// TranslateStream performs streaming translation
func (c *Client) TranslateStream(ctx context.Context, req *TranslateRequest, onChunk func(StreamChunk)) error {
	prompt := buildPrompt(req.Text, req.From, req.To, req.Prompt)
	model := getDefaultModel(req.Model)

	input := &bedrockruntime.ConverseStreamInput{
		ModelId: aws.String(model),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: prompt},
				},
			},
		},
	}

	output, err := c.runtime.ConverseStream(ctx, input)
	if err != nil {
		return fmt.Errorf("bedrock converse stream failed: %w", err)
	}

	stream := output.GetStream()
	defer stream.Close()

	for event := range stream.Events() {
		switch v := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			if delta, ok := v.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
				onChunk(StreamChunk{Content: delta.Value})
			}
		case *types.ConverseStreamOutputMemberMessageStop:
			onChunk(StreamChunk{Done: true})
		}
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	return nil
}

// MarshalChunk converts StreamChunk to JSON bytes
func MarshalChunk(chunk StreamChunk) []byte {
	data, _ := json.Marshal(chunk)
	return data
}
