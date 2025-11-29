---
title: Multi-Modality Support
description: Send images, audio, and video to agents alongside text
---

# Multi-Modality Support

Hector agents support **multi-modal inputs**, allowing you to send images, audio, video, and other media types alongside text messages. This enables powerful use cases like image analysis, document understanding, visual question answering, and more.

## Overview

Multi-modality support in Hector follows the [A2A Protocol v0.3.0](https://a2a-protocol.org) specification, using `FilePart` messages to represent media content. All LLM providers (OpenAI, Anthropic, Gemini, Ollama) automatically handle multi-modal content when supported by their models.

### Supported Media Types

| Media Type | Supported Formats | Provider Support |
|------------|------------------|-----------------|
| **Images** | JPEG, PNG, GIF, WebP | ✅ All providers |
| **Video** | MP4, AVI, MOV (via URIs) | ✅ Gemini |
| **Audio** | WAV, MP3 (via URIs) | ✅ Gemini |

**Note:** Image support is universal across all providers. Video and audio support varies by provider capabilities.

---

## Quick Start

### Sending Images via A2A Protocol

**Using HTTP/REST:**

```bash
curl -X POST http://localhost:8080/v1/agents/assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "parts": [
        {
          "text": "What is in this image?"
        },
        {
          "file": {
            "file_with_bytes": "<base64-encoded-image>",
            "media_type": "image/jpeg",
            "name": "photo.jpg"
          }
        }
      ]
    }
  }'
```

**Using File URI:**

```json
{
  "message": {
    "role": "ROLE_USER",
    "parts": [
      {
        "text": "Analyze this image"
      },
      {
        "file": {
          "file_with_uri": "https://example.com/image.jpg",
          "media_type": "image/jpeg",
          "name": "image.jpg"
        }
      }
    ]
  }
}
```

### Programmatic API

```go
import (
    "github.com/kadirpekel/hector/pkg/a2a/pb"
    "github.com/kadirpekel/hector/pkg/agent"
)

// Create message with image
msg := &pb.Message{
    Role: pb.Role_ROLE_USER,
    Parts: []*pb.Part{
        {
            Part: &pb.Part_Text{
                Text: "What's in this image?",
            },
        },
        {
            Part: &pb.Part_File{
                File: &pb.FilePart{
                    File: &pb.FilePart_FileWithBytes{
                        FileWithBytes: imageBytes,
                    },
                    MediaType: "image/jpeg",
                    Name:      "photo.jpg",
                },
            },
        },
    },
}

// Send to agent
response, err := agent.SendMessage(ctx, msg)
```

---

## Configuration

### Agent Card Configuration

Configure supported input/output modes in your agent's A2A card:

```yaml
agents:
  vision_assistant:
    name: "Vision Assistant"
    llm: "gpt-4o"
    
    a2a:
      version: "0.3.0"
      input_modes:
        - "text/plain"
        - "application/json"
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
      output_modes:
        - "text/plain"
        - "application/json"
```

**Default Input Modes:**

If not specified, agents automatically include these image types:
- `text/plain`
- `application/json`
- `image/jpeg`
- `image/png`
- `image/gif`
- `image/webp`

---

## LLM Provider Support

### OpenAI

**Supported Models:**
- GPT-4o, GPT-4o-mini (vision-capable)
- GPT-4 Turbo with vision

**Features:**
- ✅ Direct HTTP/HTTPS image URLs
- ✅ Base64-encoded images (data URIs)
- ✅ Maximum image size: 20MB
- ✅ Supports JPEG, PNG, GIF, WebP

**Example:**

```yaml
llms:
  vision:
    type: "openai"
    model: "gpt-4o"  # Vision-capable model
    api_key: "${OPENAI_API_KEY}"

agents:
  vision_assistant:
    llm: "vision"
```

**URI Support:**
OpenAI supports direct image URLs. Simply provide the URL in `file_with_uri`:

```json
{
  "file": {
    "file_with_uri": "https://example.com/image.jpg",
    "media_type": "image/jpeg"
  }
}
```

### Anthropic (Claude)

**Supported Models:**
- Claude Sonnet 4 (vision-capable)
- Claude Opus 4 (vision-capable)

**Features:**
- ✅ Base64-encoded images only
- ❌ Image URLs not supported (must download first)
- ✅ Maximum image size: 5MB
- ✅ Supports JPEG, PNG, GIF, WebP

**Example:**

```yaml
llms:
  claude_vision:
    type: "anthropic"
    model: "claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"

agents:
  vision_assistant:
    llm: "claude_vision"
```

**Important:** Anthropic requires base64-encoded images. If you have a URL, download the image first and convert to bytes.

### Google Gemini

**Supported Models:**
- Gemini 2.0 Flash (vision-capable)
- Gemini Pro (vision-capable)

**Features:**
- ✅ Google Cloud Storage URIs (`gs://`)
- ✅ Base64-encoded images (inline data)
- ✅ Video support (via URIs)
- ✅ Audio support (via URIs)
- ✅ Maximum inline size: 20MB
- ✅ Supports JPEG, PNG, GIF, WebP, MP4, AVI, MOV, WAV, MP3

**Example:**

```yaml
llms:
  gemini_vision:
    type: "gemini"
    model: "gemini-2.0-flash-exp"
    api_key: "${GEMINI_API_KEY}"

agents:
  vision_assistant:
    llm: "gemini_vision"
```

**URI Support:**
Gemini supports Google Cloud Storage URIs and some HTTP URLs (may require File API upload):

```json
{
  "file": {
    "file_with_uri": "gs://bucket/image.jpg",
    "media_type": "image/jpeg"
  }
}
```

### Ollama

**Supported Models:**
- qwen3 (vision-capable)
- Other vision-capable models

**Features:**
- ✅ Base64-encoded images only
- ❌ Image URLs not supported (must download first)
- ✅ Maximum image size: 20MB
- ✅ Supports JPEG, PNG, GIF, WebP

**Example:**

```yaml
llms:
  local_vision:
    type: "ollama"
    model: "qwen3"
    host: "http://localhost:11434"

agents:
  vision_assistant:
    llm: "local_vision"
```

---

## FilePart Message Format

The A2A Protocol defines `FilePart` for multi-modal content:

```protobuf
message FilePart {
  oneof file {
    string file_with_uri = 1;    // HTTP/HTTPS URL or GCS URI
    bytes file_with_bytes = 2;   // Base64-encoded data
  }
  string media_type = 3;         // MIME type (e.g., "image/jpeg")
  string name = 4;               // Optional filename
}
```

### Field Details

**`file_with_uri`** (string):
- HTTP/HTTPS URL: `https://example.com/image.jpg`
- Google Cloud Storage URI: `gs://bucket/image.jpg`
- Supported by: OpenAI, Gemini

**`file_with_bytes`** (bytes):
- Base64-encoded image data
- Supported by: All providers
- Recommended for: Anthropic, Ollama

**`media_type`** (string):
- MIME type identifier
- Examples: `image/jpeg`, `image/png`, `image/gif`, `image/webp`
- Required for proper processing

**`name`** (string, optional):
- Filename for reference
- Used in tool results and logging

---

## Use Cases

### Image Analysis

Analyze images and answer questions about their content:

```json
{
  "message": {
    "role": "ROLE_USER",
    "parts": [
      {
        "text": "What objects are in this image?"
      },
      {
        "file": {
          "file_with_bytes": "<base64-image>",
          "media_type": "image/jpeg"
        }
      }
    ]
  }
}
```

### Document Understanding

Extract text and information from images of documents:

```
User: [sends image of invoice]
Agent: This invoice shows:
- Invoice #: INV-2024-001
- Amount: $1,250.00
- Due date: 2024-12-31
```

### Visual Question Answering

Answer questions about image content:

```
User: [sends photo] "What color is the car?"
Agent: The car in the image is red.
```

### Multi-Modal Conversations

Combine text and images in conversation:

```
User: [sends diagram] "Explain this architecture"
Agent: This diagram shows a microservices architecture with...
User: [sends updated diagram] "What changed?"
Agent: The new version adds a load balancer and...
```

---

## Best Practices

### 1. Choose the Right Provider

| Use Case | Recommended Provider | Reason |
|----------|---------------------|--------|
| Image URLs | OpenAI | Direct URL support |
| Large images (>5MB) | OpenAI, Gemini, Ollama | Higher size limits |
| Video/Audio | Gemini | Multi-modal support |
| Cost-sensitive | OpenAI GPT-4o-mini | Lower cost |
| Local processing | Ollama | No API costs |

### 2. Optimize Image Sizes

- **Resize images** before sending (most models work well with 1024x1024)
- **Compress images** to reduce payload size
- **Use appropriate formats** (JPEG for photos, PNG for graphics)

### 3. Handle Provider Limitations

**Anthropic URI Limitation:**

If using Anthropic with image URLs, download and convert first:

```go
// Download image
resp, err := http.Get(imageURL)
if err != nil {
    return err
}
defer resp.Body.Close()

imageBytes, err := io.ReadAll(resp.Body)
if err != nil {
    return err
}

// Use file_with_bytes instead
filePart := &pb.FilePart{
    File: &pb.FilePart_FileWithBytes{
        FileWithBytes: imageBytes,
    },
    MediaType: "image/jpeg",
}
```

### 4. Set Media Types Correctly

Always specify `media_type` for proper processing:

```json
{
  "file": {
    "file_with_bytes": "<base64>",
    "media_type": "image/jpeg"  // ✅ Required
  }
}
```

### 5. Error Handling

Handle cases where images are skipped:

- **Oversized images**: Check size limits (5MB for Anthropic, 20MB for others)
- **Unsupported formats**: Ensure media type starts with `image/`
- **Invalid URIs**: Verify URLs are accessible

---

## Size Limits

| Provider | Maximum Size | Notes |
|----------|--------------|-------|
| OpenAI | 20MB | Both URIs and base64 |
| Anthropic | 5MB | Base64 only |
| Gemini | 20MB | Inline data; URIs vary |
| Ollama | 20MB | Base64 only |

**Recommendation:** Keep images under 5MB for maximum compatibility.

---

## Troubleshooting

### Images Not Being Processed

**Check:**
1. ✅ Model supports vision (e.g., `gpt-4o`, not `gpt-3.5-turbo`)
2. ✅ Media type is set correctly (`image/jpeg`, `image/png`, etc.)
3. ✅ Image size is within limits
4. ✅ Provider supports your input method (URI vs bytes)

### Anthropic URI Errors

**Problem:** Anthropic doesn't support image URLs directly.

**Solution:** Download image and use `file_with_bytes`:

```go
// Download first
imageBytes := downloadImage(url)

// Then use bytes
filePart := &pb.FilePart{
    File: &pb.FilePart_FileWithBytes{
        FileWithBytes: imageBytes,
    },
    MediaType: detectMediaType(imageBytes),
}
```

### Gemini URI Limitations

**Problem:** Standard HTTP URLs may not work with Gemini.

**Solution:** Use Google Cloud Storage URIs or convert to base64:

```json
{
  "file": {
    "file_with_uri": "gs://my-bucket/image.jpg"  // ✅ Works
    // OR
    "file_with_bytes": "<base64>"  // ✅ Always works
  }
}
```

---

## Examples

### Complete Configuration

```yaml
llms:
  vision_llm:
    type: "openai"
    model: "gpt-4o"
    api_key: "${OPENAI_API_KEY}"

agents:
  vision_assistant:
    name: "Vision Assistant"
    llm: "vision_llm"
    
    a2a:
      version: "0.3.0"
      input_modes:
        - "text/plain"
        - "image/jpeg"
        - "image/png"
        - "image/gif"
        - "image/webp"
    
    tools:
    
    prompt:
      system_role: |
        You are a vision assistant that can analyze images
        and answer questions about their content.
```

### REST API Example

```bash
# Send image via REST
curl -X POST http://localhost:8080/v1/agents/vision_assistant/message:send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "ROLE_USER",
      "parts": [
        {
          "text": "Describe this image"
        },
        {
          "file": {
            "file_with_uri": "https://example.com/photo.jpg",
            "media_type": "image/jpeg",
            "name": "photo.jpg"
          }
        }
      ]
    }
  }'
```

### Programmatic Example

```go
package main

import (
    "context"
    "encoding/base64"
    "io"
    "net/http"
    
    "github.com/kadirpekel/hector/pkg/a2a/pb"
    "github.com/kadirpekel/hector/pkg/agent"
)

func sendImageMessage(ctx context.Context, agent *agent.Agent, imageURL string) error {
    // Download image
    resp, err := http.Get(imageURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    imageBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }
    
    // Create message with image
    msg := &pb.Message{
        Role: pb.Role_ROLE_USER,
        Parts: []*pb.Part{
            {
                Part: &pb.Part_Text{
                    Text: "What's in this image?",
                },
            },
            {
                Part: &pb.Part_File{
                    File: &pb.FilePart{
                        File: &pb.FilePart_FileWithBytes{
                            FileWithBytes: imageBytes,
                        },
                        MediaType: "image/jpeg",
                        Name:      "image.jpg",
                    },
                },
            },
        },
    }
    
    // Send to agent
    response, err := agent.SendMessage(ctx, msg)
    if err != nil {
        return err
    }
    
    // Process response
    // ...
    
    return nil
}
```

---

## Next Steps

- **[Tools](tools.md)** - Learn about available tools
- **[LLM Providers](llm-providers.md)** - Configure vision-capable models
- **[A2A Protocol](../reference/a2a-protocol.md)** - Understand FilePart message format
- **[Configuration Reference](../reference/configuration.md)** - Complete configuration options

---

## Related Topics

- **[Tools](tools.md)** - Vision tools and capabilities
- **[LLM Providers](llm-providers.md)** - Provider-specific multi-modality support
- **[A2A Protocol](../reference/a2a-protocol.md)** - Protocol specification
- **[Programmatic API](programmatic-api.md)** - Using multi-modality in code

