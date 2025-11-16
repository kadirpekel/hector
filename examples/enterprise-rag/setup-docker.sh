#!/bin/bash
# Setup script for Enterprise RAG Lab Environment
# This script initializes the complete RAG system with all dependencies

set -e

echo "🚀 Setting up Enterprise RAG Lab Environment..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start all services
echo "📦 Starting Docker services..."
docker-compose up -d

echo ""
echo "⏳ Waiting for services to be healthy..."
echo "   (This may take 1-2 minutes for Ollama to be ready)"

# Wait for Ollama to be ready
echo -n "   Waiting for Ollama..."
until docker exec lab-ollama curl -f http://localhost:11434/api/tags > /dev/null 2>&1; do
    echo -n "."
    sleep 2
done
echo " ✅"

# Pull embedding model
echo ""
echo "📥 Downloading embedding model (nomic-embed-text)..."
echo "   This may take a few minutes on first run..."
docker exec lab-ollama ollama pull nomic-embed-text

# Pull LLM model (qwen3)
echo ""
echo "📥 Downloading LLM model (qwen3)..."
echo "   This may take several minutes on first run..."
docker exec lab-ollama ollama pull qwen3

echo ""
echo "✅ Setup complete!"
echo ""
echo "📊 Service Status:"
echo "   - Qdrant:      http://localhost:6334/dashboard"
echo "   - Ollama:      http://localhost:11434"
echo "   - PostgreSQL:  localhost:5433"
echo "   - Wiki API:    http://localhost:8081/health"
echo "   - Hector:      http://localhost:8080/health"
echo "   - Prometheus:  http://localhost:9090"
echo ""
echo "🧪 Test the system:"
echo "   docker exec lab-hector hector call \"What are our password requirements?\" \\"
echo "     --agent enterprise_assistant \\"
echo "     --config /etc/hector/config.yaml"
echo ""
echo "📖 For more information, see examples/enterprise-rag/README.md"

