package tools

import (
	"context"
	"testing"
	"time"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewGenerateImageToolWithConfig(t *testing.T) {
	toolConfig := &config.ToolConfig{
		Type: "generate_image",
		Config: map[string]interface{}{
			"api_key": "test-api-key",
			"model":   "dall-e-2",
			"size":    "512x512",
			"quality": "hd",
			"style":   "natural",
			"timeout": "45s",
		},
	}

	tool, err := NewGenerateImageToolWithConfig("generate_image", toolConfig)
	assert.NoError(t, err)
	assert.NotNil(t, tool)

	info := tool.GetInfo()
	assert.Equal(t, "generate_image", info.Name)
	assert.Equal(t, "generate_image", tool.GetName())

	// Verify defaults are overridden
	genTool, ok := tool.(*GenerateImageTool)
	assert.True(t, ok)
	assert.Equal(t, "test-api-key", genTool.config.APIKey)
	assert.Equal(t, "dall-e-2", genTool.config.Model)
	assert.Equal(t, "512x512", genTool.config.Size)
	assert.Equal(t, "hd", genTool.config.Quality)
	assert.Equal(t, "natural", genTool.config.Style)
	assert.Equal(t, 45*time.Second, genTool.config.Timeout)
}

func TestNewGenerateImageToolWithDefaults(t *testing.T) {
	toolConfig := &config.ToolConfig{
		Type:   "generate_image",
		Config: map[string]interface{}{},
	}

	tool, err := NewGenerateImageToolWithConfig("generate_image", toolConfig)
	assert.NoError(t, err)
	assert.NotNil(t, tool)

	genTool, ok := tool.(*GenerateImageTool)
	assert.True(t, ok)
	assert.Equal(t, "dall-e-3", genTool.config.Model)
	assert.Equal(t, "1024x1024", genTool.config.Size)
	assert.Equal(t, "standard", genTool.config.Quality)
	assert.Equal(t, "vivid", genTool.config.Style)
	assert.Equal(t, 60*time.Second, genTool.config.Timeout)
}

func TestGenerateImageTool_Execute_MissingPrompt(t *testing.T) {
	toolConfig := &config.ToolConfig{
		Type: "generate_image",
		Config: map[string]interface{}{
			"api_key": "test-key",
		},
	}
	tool, _ := NewGenerateImageToolWithConfig("generate_image", toolConfig)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "prompt is required", result.Error)
}

func TestNewScreenshotPageToolWithConfig(t *testing.T) {
	toolConfig := &config.ToolConfig{
		Type: "screenshot_page",
		Config: map[string]interface{}{
			"timeout": "1m",
		},
	}

	tool, err := NewScreenshotPageToolWithConfig("screenshot_page", toolConfig)
	assert.NoError(t, err)
	assert.NotNil(t, tool)

	info := tool.GetInfo()
	assert.Equal(t, "screenshot_page", info.Name)

	screenTool, ok := tool.(*ScreenshotPageTool)
	assert.True(t, ok)
	assert.Equal(t, 60*time.Second, screenTool.config.Timeout)
}

func TestScreenshotPageTool_Execute_Placeholder(t *testing.T) {
	toolConfig := &config.ToolConfig{
		Type: "screenshot_page",
	}
	tool, _ := NewScreenshotPageToolWithConfig("screenshot_page", toolConfig)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"url": "https://example.com",
	})
	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not yet implemented")
}
