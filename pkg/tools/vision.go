package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/httpclient"
)

type GenerateImageTool struct {
	config     *GenerateImageConfig
	httpClient *httpclient.Client
}

type GenerateImageConfig struct {
	APIKey  string
	Model   string
	Size    string
	Quality string
	Style   string
	Timeout time.Duration
}

func NewGenerateImageTool(cfg *GenerateImageConfig) *GenerateImageTool {
	httpClientConfig := &http.Client{
		Timeout: cfg.Timeout,
	}

	hc := httpclient.New(
		httpclient.WithHTTPClient(httpClientConfig),
		httpclient.WithMaxRetries(3),
	)

	return &GenerateImageTool{
		config:     cfg,
		httpClient: hc,
	}
}

func NewGenerateImageToolWithConfig(toolName string, toolConfig *config.ToolConfig) (Tool, error) {
	apiKey, _ := toolConfig.Config["api_key"].(string)
	if apiKey == "" {
		// Fallback to environment variable
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	model, _ := toolConfig.Config["model"].(string)
	if model == "" {
		model = "dall-e-3"
	}

	size, _ := toolConfig.Config["size"].(string)
	if size == "" {
		size = "1024x1024"
	}

	quality, _ := toolConfig.Config["quality"].(string)
	if quality == "" {
		quality = "standard"
	}

	style, _ := toolConfig.Config["style"].(string)
	if style == "" {
		style = "vivid"
	}

	timeoutStr, _ := toolConfig.Config["timeout"].(string)
	timeout := 60 * time.Second
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	}

	cfg := &GenerateImageConfig{
		APIKey:  apiKey,
		Model:   model,
		Size:    size,
		Quality: quality,
		Style:   style,
		Timeout: timeout,
	}

	return NewGenerateImageTool(cfg), nil
}

func (t *GenerateImageTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "generate_image",
		Description: "Generate an image from a text prompt using DALL-E 3",
		Parameters: []ToolParameter{
			{
				Name:        "prompt",
				Type:        "string",
				Description: "The text prompt to generate an image for",
				Required:    true,
			},
			{
				Name:        "size",
				Type:        "string",
				Description: "The size of the generated image (e.g., 1024x1024)",
				Required:    false,
				Default:     t.config.Size,
			},
			{
				Name:        "quality",
				Type:        "string",
				Description: "The quality of the image (standard or hd)",
				Required:    false,
				Enum:        []string{"standard", "hd"},
				Default:     t.config.Quality,
			},
			{
				Name:        "style",
				Type:        "string",
				Description: "The style of the image (vivid or natural)",
				Required:    false,
				Enum:        []string{"vivid", "natural"},
				Default:     t.config.Style,
			},
		},
		ServerURL: "local",
	}
}

func (t *GenerateImageTool) GetName() string {
	return "generate_image"
}

func (t *GenerateImageTool) GetDescription() string {
	return "Generate an image from a text prompt using DALL-E 3"
}

type OpenAIImageRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Quality string `json:"quality,omitempty"`
	Style   string `json:"style,omitempty"`
}

type OpenAIImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func (t *GenerateImageTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	prompt, ok := args["prompt"].(string)
	if !ok || prompt == "" {
		return ToolResult{
			Success:       false,
			Error:         "prompt is required",
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("prompt is required")
	}

	size := t.config.Size
	if s, ok := args["size"].(string); ok && s != "" {
		size = s
	}

	quality := t.config.Quality
	if q, ok := args["quality"].(string); ok && q != "" {
		quality = q
	}

	style := t.config.Style
	if s, ok := args["style"].(string); ok && s != "" {
		style = s
	}

	if t.config.APIKey == "" {
		// Try to fallback to env var if not set in config
		// But for now, just error
		return ToolResult{
			Success:       false,
			Error:         "API key is not configured for generate_image tool",
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("API key is not configured")
	}

	reqBody := OpenAIImageRequest{
		Model:   t.config.Model,
		Prompt:  prompt,
		N:       1,
		Size:    size,
		Quality: quality,
		Style:   style,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to marshal request: %v", err),
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/images/generations", bytes.NewReader(jsonData))
	if err != nil {
		return ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to create request: %v", err),
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.config.APIKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("request failed: %v", err),
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var response OpenAIImageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to parse response: %v", err),
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, err
	}

	if response.Error != nil {
		return ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("OpenAI API error: %s", response.Error.Message),
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Data) == 0 {
		return ToolResult{
			Success:       false,
			Error:         "no image data returned",
			ToolName:      "generate_image",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("no image data returned")
	}

	imageURL := response.Data[0].URL
	revisedPrompt := response.Data[0].RevisedPrompt

	return ToolResult{
		Success:       true,
		Content:       fmt.Sprintf("Image generated successfully: %s", imageURL),
		ToolName:      "generate_image",
		ExecutionTime: time.Since(start),
		Output:        imageURL,
		Metadata: map[string]interface{}{
			"url":            imageURL,
			"revised_prompt": revisedPrompt,
		},
	}, nil
}

// ScreenshotPageTool is a PLACEHOLDER for future headless browser integration.
// Currently not implemented - requires integration with Chrome DevTools Protocol
// or similar headless browser solution (e.g., chromedp, playwright).
//
// TODO: Implement using a headless browser library
type ScreenshotPageTool struct {
	config *ScreenshotPageConfig
}

type ScreenshotPageConfig struct {
	Timeout time.Duration
}

func NewScreenshotPageTool(cfg *ScreenshotPageConfig) *ScreenshotPageTool {
	return &ScreenshotPageTool{
		config: cfg,
	}
}

func NewScreenshotPageToolWithConfig(toolName string, toolConfig *config.ToolConfig) (Tool, error) {
	timeoutStr, _ := toolConfig.Config["timeout"].(string)
	timeout := 30 * time.Second
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	}

	cfg := &ScreenshotPageConfig{
		Timeout: timeout,
	}

	return NewScreenshotPageTool(cfg), nil
}

func (t *ScreenshotPageTool) GetInfo() ToolInfo {
	return ToolInfo{
		Name:        "screenshot_page",
		Description: "[NOT IMPLEMENTED] Take a screenshot of a web page. This tool is a placeholder and will return an error when called. Requires headless browser integration to be completed.",
		Parameters: []ToolParameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to screenshot",
				Required:    true,
			},
		},
		ServerURL: "local",
	}
}

func (t *ScreenshotPageTool) GetName() string {
	return "screenshot_page"
}

func (t *ScreenshotPageTool) GetDescription() string {
	return "[NOT IMPLEMENTED] Take a screenshot of a web page (requires headless browser)"
}

func (t *ScreenshotPageTool) Execute(ctx context.Context, args map[string]interface{}) (ToolResult, error) {
	start := time.Now()

	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return ToolResult{
			Success:       false,
			Error:         "url is required",
			ToolName:      "screenshot_page",
			ExecutionTime: time.Since(start),
		}, fmt.Errorf("url is required")
	}

	// Placeholder implementation
	return ToolResult{
		Success:       false,
		Error:         "Screenshot capability is not yet implemented (requires headless browser configuration)",
		ToolName:      "screenshot_page",
		ExecutionTime: time.Since(start),
		Metadata: map[string]interface{}{
			"url": urlStr,
		},
	}, fmt.Errorf("not implemented")
}
