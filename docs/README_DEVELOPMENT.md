# Hector Documentation - Local Development

## Prerequisites

- Ruby 3.1+
- Bundler

## Setup

1. Install dependencies:
   ```bash
   cd docs
   bundle install
   ```

2. Run locally:
   ```bash
   bundle exec jekyll serve --livereload
   ```

3. Open http://localhost:4000/hector

## Building

```bash
bundle exec jekyll build
```

## Deployment

GitHub Pages automatically deploys when changes are pushed to `main` branch in the `docs/` directory.

## Custom Domain

If using a custom domain, update the `CNAME` file and configure DNS:

- A record: 185.199.108.153, 185.199.109.153, 185.199.110.153, 185.199.111.153
- AAAA record: 2606:50c0:8000::153, 2606:50c0:8001::153, 2606:50c0:8002::153, 2606:50c0:8003::153
