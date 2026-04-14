# Deployments

This directory contains deployment configurations and infrastructure setup files for Hector.

## Files

- **docker-compose.docling.yaml** - Docker Compose setup for running Hector with Docling document parsing
- **docker-compose.observability.yaml** - Full observability stack (Prometheus, Jaeger, Grafana)
- **docker-compose.config-providers.yaml** - Configuration providers for testing (Consul, Etcd, ZooKeeper)
- **prometheus.yaml** - Prometheus configuration file

## Usage

### Observability Stack

Start the full observability stack:

```bash
docker-compose -f deployments/docker-compose.observability.yaml up -d
```

This starts:
- **Jaeger** on port 16686 (UI) and 4317 (OTLP gRPC)
- **Prometheus** on port 9090
- **Grafana** on port 3000 (login: admin/Dev12345)

### Docling Integration

Start Hector with Docling:

```bash
docker-compose -f deployments/docker-compose.docling.yaml up -d
```

### Configuration Providers

Start configuration providers for testing:

```bash
docker-compose -f deployments/docker-compose.config-providers.yaml up -d
```

## Notes

- All docker-compose files use relative paths from this directory
- Grafana dashboards and datasources are located in `../grafana/`
- Prometheus configuration is in `prometheus.yaml` in this directory

