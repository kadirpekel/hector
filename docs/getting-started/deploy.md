# Deploy Hector

Get your own Hector instance running in under a minute. Pick a cloud provider below, log in with your existing account, fill in a couple of fields, and you're done — full Hector with embedded Studio UI at an auto-assigned URL.

Every deployed instance is **self-contained**: agents, tools, Studio UI, admin API, and A2A protocol all in one binary. No external dependencies required.

---

## Cloud Providers

### Railway

One-click deploy with scale-to-zero. SQLite for storage — zero setup.

| | |
|---|---|
| **Domain** | `*.up.railway.app` |
| **Auth** | GitHub OAuth |
| **Database** | SQLite (built-in) |
| **Free Tier** | Trial credits |
| **Scale to Zero** | Yes |

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/new/template?template=https://github.com/verikod/hector)

??? info "What gets provisioned"
    - Docker container from `ghcr.io/verikod/hector:latest`
    - Health checks at `/health`
    - You set `HECTOR_AUTH_SECRET` during deploy
    - Recipe: [`railway.json`](https://github.com/verikod/hector/blob/main/railway.json)

---

### Render

Blueprint-based deploy with persistent disk for SQLite.

| | |
|---|---|
| **Domain** | `*.onrender.com` |
| **Auth** | GitHub / Google / GitLab OAuth |
| **Database** | SQLite (persistent disk) |
| **Free Tier** | Yes |
| **Scale to Zero** | Paid plans |

[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/verikod/hector)

??? info "What gets provisioned"
    - Docker web service with health checks
    - 1 GB persistent disk at `/data` for SQLite
    - You set `HECTOR_AUTH_SECRET` during deploy
    - Recipe: [`render.yaml`](https://github.com/verikod/hector/blob/main/render.yaml)

---

### Heroku

Classic one-click deploy. SQLite for storage.

| | |
|---|---|
| **Domain** | `*.herokuapp.com` |
| **Auth** | Heroku account |
| **Database** | SQLite (built-in) |
| **Free Tier** | Eco ($5/mo) |
| **Scale to Zero** | No |

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/verikod/hector)

??? info "What gets provisioned"
    - Container stack with Hector Docker image
    - You set `HECTOR_AUTH_SECRET` during deploy
    - Recipe: [`app.json`](https://github.com/verikod/hector/blob/main/app.json)

---

### Fly.io

Edge deployment with auto-stop/start machines and global regions.

| | |
|---|---|
| **Domain** | `*.fly.dev` |
| **Auth** | Fly.io account |
| **Database** | SQLite (volume-backed) |
| **Free Tier** | 3 shared VMs |
| **Scale to Zero** | Yes |

```bash
# One-command deploy
fly launch --config deploy/fly/fly.toml
fly secrets set HECTOR_AUTH_SECRET=your-secret-here
```

??? info "What gets provisioned"
    - Shared CPU machine (256MB) in your chosen region
    - Auto-stop after idle, auto-start on request
    - Health checks at `/health`
    - Recipe: [`deploy/fly/fly.toml`](https://github.com/verikod/hector/blob/main/deploy/fly/fly.toml)

---

### Docker Compose (Self-Hosted)

Run Hector on your own infrastructure. SQLite with persistent volume.

| | |
|---|---|
| **Domain** | Your own |
| **Auth** | N/A |
| **Database** | SQLite (volume-backed) |
| **Free** | Yes |
| **Scale to Zero** | N/A |

```bash
export HECTOR_AUTH_SECRET=your-secret-here
curl -O https://raw.githubusercontent.com/verikod/hector/main/deploy/docker-compose/docker-compose.yaml
docker compose up -d
```

Open `http://localhost:8080` — Hector with embedded Studio UI is ready.

??? info "What gets provisioned"
    - Hector container (`ghcr.io/verikod/hector:latest`)
    - Persistent volume for SQLite data
    - Recipe: [`deploy/docker-compose/docker-compose.yaml`](https://github.com/verikod/hector/blob/main/deploy/docker-compose/docker-compose.yaml)

---

## After Deployment

Once your instance is running:

1. **Open the URL** provided by your cloud platform (e.g., `my-hector.up.railway.app`)
2. **Studio UI** loads automatically at the root `/`
3. **Enter your admin secret** — the one you set during deployment
4. **Configure LLM providers and agents** via the visual editor or upload a YAML config

Your instance includes everything: Studio UI, Admin API (`/admin/`), A2A protocol (`/agents/`), webhooks (`/webhooks/`), and health checks (`/health`).

---

## Add a Provider

Anyone can contribute a deploy recipe for a new cloud provider. See the [Deploy Recipe Contributing Guide](https://github.com/verikod/hector/blob/main/deploy/CONTRIBUTING.md) for details.

Providers that require their config at the repo root (e.g., Railway, Render, Heroku) place the recipe file at the top level. Each provider also gets a `deploy/<provider>/README.md` for documentation.
