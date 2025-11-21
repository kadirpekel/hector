package llms

import (
	"net/http"
	"strings"
)

// detectImageMediaType detects the MIME type of an image from its bytes
// by examining the magic number/signature at the start of the file.
// This function is used by all LLM providers (Anthropic, OpenAI, Gemini, Ollama)
// to automatically detect image format when mediaType is not provided.
func detectImageMediaType(data []byte) string {
	if len(data) == 0 {
		return "image/jpeg" // Default fallback
	}

	// Use http.DetectContentType which examines magic numbers
	detected := http.DetectContentType(data)

	// Ensure it's an image type, otherwise default to JPEG
	if strings.HasPrefix(detected, "image/") {
		return detected
	}

	// Manual detection for cases where DetectContentType might not work
	// Check magic numbers directly
	if len(data) >= 4 {
		// PNG: 89 50 4E 47
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
			return "image/png"
		}
		// JPEG: FF D8 FF
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		// GIF: 47 49 46 38 (GIF8)
		if len(data) >= 6 && data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
			return "image/gif"
		}
		// WebP: RIFF...WEBP
		if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
			data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
			return "image/webp"
		}
	}

	// Default fallback
	return "image/jpeg"
}
