# Stockyard Strongbox

**CI/CD secrets manager — store encrypted env vars, expose via short-lived tokens**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted developer tools.

## Quick Start

```bash
docker run -p 9070:9070 -v strongbox_data:/data ghcr.io/stockyard-dev/stockyard-strongbox
```

Or with docker-compose:

```bash
docker-compose up -d
```

Open `http://localhost:9070` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9070` | HTTP port |
| `DATA_DIR` | `./data` | SQLite database directory |
| `STRONGBOX_LICENSE_KEY` | *(empty)* | Pro license key |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 2 vaults, 10 secrets | Unlimited vaults and secrets |
| Price | Free | $2.99/mo |

Get a Pro license at [stockyard.dev/tools/](https://stockyard.dev/tools/).

## Category

Developer Tools

## License

Apache 2.0
