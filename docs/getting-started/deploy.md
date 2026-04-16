# Deploy Hector

Get your own Hector instance running in under a minute. Pick a cloud provider below, log in with your existing account, fill in a couple of fields, and you're done — full Hector with embedded Studio UI at an auto-assigned URL.

Every deployed instance is **self-contained**: agents, tools, Studio UI, admin API, and A2A protocol all in one binary. No external dependencies required.

---

## Cloud Providers

### Railway

One-click deploy with auto-generated secrets, optional PostgreSQL plugin, and scale-to-zero.

| | |
|---|---|
| **Domain** | `*.up.railway.app` |
| **Auth** | GitHub OAuth |
| **Database** | PostgreSQL plugin (add from dashboard) |
| **Free Tier** | Trial credits |
| **Scale to Zero** | Yes |

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/hector?referralCode=hector)

??? info "What gets provisioned"
    - Docker container from `ghcr.io/verikod/hector:latest`
    - `HECTOR_AUTH_SECRET` auto-generated
    - Optional: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
    - Health checks at `/health`
    - Recipe: [`railway.json`](https://github.com/verikod/hector/blob/main/railway.json)

---

### Render

Blueprint-based deploy with auto-provisioned free PostgreSQL and auto-generated admin secret.

| | |
|---|---|
| **Domain** | `*.onrender.com` |
| **Auth** | GitHub / Google / GitLab OAuth |
| **Database** | PostgreSQL (auto-provisioned, free plan) |
| **Free Tier** | Yes |
| **Scale to Zero** | Paid plans |

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/verikod/hector)

??? info "What gets provisioned"
    - Docker web service with health checks
    - PostgreSQL database (free plan, auto-connected)
    - `HECTOR_AUTH_SECRET` auto-generated
    - Optional: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
    - Recipe: [`render.yaml`](https://github.com/verikod/hector/blob/main/render.yaml)

---

### Heroku

Classic one-click deploy with managed PostgreSQL addon.

| | |
|---|---|
| **Domain** | `*.herokuapp.com` |
| **Auth** | Heroku account |
| **Database** | PostgreSQL (Essential-0 addon) |
| **Free Tier** | Eco ($5/mo) |
| **Scale to Zero** | No |

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/verikod/hector)

??? info "What gets provisioned"
    - Container stack with Hector Docker image
    - PostgreSQL Essential-0 addon (auto-connected)
    - `HECTOR_AUTH_SECRET` auto-generated
    - Optional: `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`
    - Recipe: [`app.json`](https://github.com/verikod/hector/blob/main/app.json)

---

### Fly.io

Edge deployment with auto-stop/start machines and global regions.

| | |
|---|---|
| **Domain** | `*.fly.dev` |
| **Auth** | Fly.io account |
| **Database** | `fly postgres create` (separate step) |
| **Free Tier** | 3 shared VMs |
| **Scale to Zero** | Yes |

```bash
# One-command deploy
fly launch --config deploy/fly/fly.toml
fly secrets set HECTOR_AUTH_SECRET=$(openssl rand -hex 32)
```

??? info "What gets provisioned"
    - Shared CPU machine (256MB) in your chosen region
    - Auto-stop after idle, auto-start on request
    - Health checks at `/health`
    - Recipe: [`deploy/fly/fly.toml`](https://github.com/verikod/hector/blob/main/deploy/fly/fly.toml)

---

### Docker Compose (Self-Hosted)

Run Hector with PostgreSQL on your own infrastructure.

| | |
|---|---|
| **Domain** | Your own |
| **Auth** | N/A |
| **Database** | PostgreSQL (included) |
| **Free** | Yes |
| **Scale to Zero** | N/A |

```bash
export HECTOR_AUTH_SECRET=$(openssl rand -hex 32)
curl -O https://raw.githubusercontent.com/verikod/hector/main/deploy/docker-compose/docker-compose.yaml
docker compose up -d
```

Open `http://localhost:8080` — Hector with embedded Studio UI is ready.

??? info "What gets provisioned"
    - Hector container (`ghcr.io/verikod/hector:latest`)
    - PostgreSQL 16 container
    - Persistent volumes for data and database
    - Recipe: [`deploy/docker-compose/docker-compose.yaml`](https://github.com/verikod/hector/blob/main/deploy/docker-compose/docker-compose.yaml)

---

## After Deployment

Once your instance is running:

1. **Open the URL** provided by your cloud platform (e.g., `my-hector.up.railway.app`)
2. **Studio UI** loads automatically at the root `/`
3. **Configure agents** via the visual editor or upload a YAML config
4. **Set LLM keys** in the environment if you haven't already

Your instance includes everything: Studio UI, Admin API (`/admin/`), A2A protocol (`/agents/`), webhooks (`/webhooks/`), and health checks (`/health`).

---

## Add a Provider

Anyone can contribute a deploy recipe for a new cloud provider. See the [Deploy Recipe Contributing Guide](https://github.com/verikod/hector/blob/main/deploy/CONTRIBUTING.md) for details.

Providers that require their config at the repo root (e.g., Railway, Render, Heroku) place the recipe file at the top level. Each provider also gets a `deploy/<provider>/README.md` for documentation.
