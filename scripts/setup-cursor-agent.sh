#!/bin/bash
#
# Setup script for Cursor-like coding assistant
# This script sets up all dependencies needed for the tutorial
#

set -e

echo "🚀 Setting up Cursor-like coding assistant..."
echo

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Step 1: Check Docker
echo "📦 Step 1/4: Checking Docker..."
if command_exists docker; then
    echo -e "${GREEN}✓${NC} Docker is installed"
else
    echo -e "${RED}✗${NC} Docker is not installed"
    echo "   Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Step 2: Start Qdrant
echo
echo "🗄️  Step 2/4: Starting Qdrant vector database..."
if docker ps | grep -q qdrant; then
    echo -e "${YELLOW}⚠${NC}  Qdrant is already running"
else
    docker run -d \
        --name qdrant \
        -p 6334:6333 \
        -p 6333:6333 \
        qdrant/qdrant
    echo -e "${GREEN}✓${NC} Qdrant started on port 6334"
fi

# Wait for Qdrant to be ready
echo "   Waiting for Qdrant to be ready..."
sleep 5
# Test both HTTP (6333) and gRPC (6334) ports
if curl -s http://localhost:6333 > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} Qdrant HTTP API ready (port 6333)"
    
    # Check if gRPC port is accessible (Hector requires this)
    if nc -z localhost 6334 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Qdrant gRPC API ready (port 6334)"
    else
        echo -e "${YELLOW}⚠${NC}  Qdrant gRPC port (6334) not accessible"
        echo "   Hector requires port 6334 (gRPC) for indexing"
        echo "   Run: docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant"
        exit 1
    fi
else
    echo -e "${RED}✗${NC} Qdrant health check failed"
    echo "   Try: curl http://localhost:6333"
    exit 1
fi

# Step 3: Check Ollama
echo
echo "🤖 Step 3/4: Checking Ollama..."
if command_exists ollama; then
    echo -e "${GREEN}✓${NC} Ollama is installed"
    
    # Check if Ollama is running
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Ollama is running"
    else
        echo -e "${YELLOW}⚠${NC}  Ollama is not running"
        echo "   Please start Ollama (it should start automatically on macOS/Linux)"
        echo "   Or run: ollama serve"
    fi
    
    # Pull embeddings model
    echo "   Pulling nomic-embed-text model..."
    if ollama list | grep -q nomic-embed-text; then
        echo -e "${GREEN}✓${NC} nomic-embed-text is already installed"
    else
        ollama pull nomic-embed-text
        echo -e "${GREEN}✓${NC} nomic-embed-text installed"
    fi
else
    echo -e "${RED}✗${NC} Ollama is not installed"
    echo "   Please install Ollama: https://ollama.ai"
    exit 1
fi

# Step 4: Check API keys
echo
echo "🔑 Step 4/4: Checking API keys..."
API_KEY_FOUND=false

if [ -n "$ANTHROPIC_API_KEY" ]; then
    echo -e "${GREEN}✓${NC} ANTHROPIC_API_KEY is set"
    API_KEY_FOUND=true
fi

if [ -n "$OPENAI_API_KEY" ]; then
    echo -e "${GREEN}✓${NC} OPENAI_API_KEY is set"
    API_KEY_FOUND=true
fi

if [ -n "$GEMINI_API_KEY" ]; then
    echo -e "${GREEN}✓${NC} GEMINI_API_KEY is set"
    API_KEY_FOUND=true
fi

if [ -n "$GLM_API_KEY" ]; then
    echo -e "${GREEN}✓${NC} GLM_API_KEY is set"
    API_KEY_FOUND=true
fi

if [ "$API_KEY_FOUND" = false ]; then
    echo -e "${YELLOW}⚠${NC}  No API keys found"
    echo
    echo "   Set one of the following environment variables:"
    echo "   • export ANTHROPIC_API_KEY='sk-ant-...'  (Recommended)"
    echo "   • export OPENAI_API_KEY='sk-...'"
    echo "   • export GEMINI_API_KEY='...'"
    echo "   • export GLM_API_KEY='...'  (For Chinese market)"
    echo
    echo "   Then update configs/cursor-production.yaml to use your chosen provider"
fi

# Summary
echo
echo "═══════════════════════════════════════════════════════════"
echo "✅ Setup complete!"
echo "═══════════════════════════════════════════════════════════"
echo
echo "Next steps:"
echo "1. Set your API key (if not done already):"
echo "   export ANTHROPIC_API_KEY='sk-ant-...'"
echo
echo "2. Edit configs/cursor-production.yaml to choose your LLM"
echo
echo "3. Start the server:"
echo "   hector serve --config configs/cursor-production.yaml"
echo
echo "4. In another terminal, start chatting:"
echo "   hector chat coding_assistant"
echo
echo "═══════════════════════════════════════════════════════════"
echo "📖 Tutorial: docs/tutorials/BUILD_YOUR_OWN_CURSOR.md"
echo "═══════════════════════════════════════════════════════════"

