# Deploy Hector on Fly.io

## Quick Start

```bash
# Install flyctl if not already installed
curl -L https://fly.io/install.sh | sh

# Clone and deploy
git clone https://github.com/verikod/hector.git
cd hector

# Launch (creates app, picks region, deploys)
fly launch --config deploy/fly/fly.toml

# Set admin secret
fly secrets set HECTOR_AUTH_SECRET=your-secret-here
```

- **Auto Domain**: `*.fly.dev`
- **Free Tier**: Yes (3 shared VMs)
- **Database**: SQLite (volume-backed)
- **Scale to Zero**: Yes (auto-stop/start)
