# Contributing Deploy Recipes

Want to add a one-click deploy option for a new cloud provider? Follow this guide.

## Structure

Providers that require their config at the repo root (e.g., Railway, Render, Heroku) place the recipe file at the top level:

```
railway.json       # Railway reads from repo root
render.yaml        # Render reads from repo root
app.json           # Heroku reads from repo root
```

Each provider also gets a folder under `deploy/` for its README and any extras:

```
deploy/
  your-provider/
    README.md        # Required: provider details and deploy instructions
```

Providers that don't auto-read from the repo (e.g., Fly.io, Docker Compose) keep everything in their `deploy/` folder:

```
deploy/
  fly/
    fly.toml
    README.md
  docker-compose/
    docker-compose.yaml
    README.md
```

## README Template

Your `README.md` must include:

```markdown
# Deploy Hector on [Provider Name]

- **Deploy URL**: `https://provider.com/deploy?template=...`
- **Auto Domain**: `*.provider.app`
- **Free Tier**: Yes / No (details)
- **Database**: How database is provisioned
- **Scale to Zero**: Yes / No
```

## Recipe Requirements

Your recipe config file must:

1. **Use the official Docker image**: `ghcr.io/verikod/hector:latest` (or build from `Dockerfile`)
2. **Auto-generate `HECTOR_AUTH_SECRET`** if the provider supports secret generation
3. **Include LLM keys as optional inputs**: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
4. **Configure health checks** at `/health`
5. **Expose port 8080** (Hector's default)

## Optional

- Provision a PostgreSQL database and wire it via `HECTOR_DATABASE` env var
- Configure scale-to-zero if the provider supports it
- Add the provider's deploy badge/button URL to your README

## Testing

Before submitting, verify:

- [ ] Deploy from scratch works with the recipe
- [ ] Hector UI loads at the root URL (`/`)
- [ ] Health check at `/health` returns 200
- [ ] Auto-generated `HECTOR_AUTH_SECRET` is set

## Submit

Open a PR adding your `deploy/<provider>/` folder. The docs page at `docs/getting-started/deploy.md` will also need an entry for your provider — include that in your PR.
