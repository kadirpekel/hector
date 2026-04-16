# Deploy Hector with Docker Compose

Self-host Hector with PostgreSQL using Docker Compose.

## Quick Start

```bash
# Generate an admin secret
export HECTOR_AUTH_SECRET=$(openssl rand -hex 32)

# Optional: set LLM provider keys
export OPENAI_API_KEY=sk-...

# Start Hector
docker compose up -d
```

Hector will be available at `http://localhost:8080` with the embedded Studio UI.

- **Auto Domain**: N/A (self-hosted)
- **Free Tier**: Yes (self-hosted)
- **Database**: PostgreSQL (included)
- **Scale to Zero**: N/A
