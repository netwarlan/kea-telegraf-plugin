# kea-telegraf-plugin

A Go binary that queries the [Kea DHCP4](https://www.isc.org/kea/) Control Agent's `statistic-get-all` API and outputs clean [InfluxDB line protocol](https://docs.influxdata.com/influxdb/v2/reference/syntax/line-protocol/) for [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/) `inputs.exec`.

## Why

Kea's stats API returns deeply nested JSON with `[[value, timestamp]]` arrays and field names containing `[]`, `.`, and `-` characters that InfluxDB 3 rejects as column names. Telegraf's built-in JSON parsers can't flatten this cleanly. This tool handles the translation — a single static binary with no external dependencies.

## Output

One global line plus one line per subnet:

```
kea_dhcp4,server=dhcp-server-01 pkt4_received=100i,pkt4_sent=50i,pkt4_ack_sent=45i,...
kea_dhcp4,server=dhcp-server-01,subnet_id=1 total_addresses=231i,assigned_addresses=10i,pool0_total_addresses=231i,...
kea_dhcp4,server=dhcp-server-01,subnet_id=2 total_addresses=11i,assigned_addresses=3i,...
```

### Field Name Cleaning

| Kea Key | Result |
|---------|--------|
| `pkt4-received` | `pkt4_received` |
| `subnet[1].total-addresses` | tag: `subnet_id=1`, field: `total_addresses` |
| `subnet[1].pool[0].total-addresses` | tag: `subnet_id=1`, field: `pool0_total_addresses` |

## Usage

```
keastats [flags]

Flags:
  -u, --url string         Kea Control Agent URL (default "http://localhost:8000/")
  -s, --server string      Server tag for line protocol output (default: hostname)
  -t, --timeout duration   HTTP request timeout (default 5s)
  -j, --json               Output raw Kea API JSON (debug mode)
  -v, --version            Print version info
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Kea unreachable or API error |
| `10` | Configuration error |

## Deployment

Three containers: Kea + keastats (init container) + stock Telegraf.

The `keastats` image copies the binary to a shared volume on startup, then exits. Telegraf calls the binary via `inputs.exec`.

### compose.yml

```yaml
services:
  kea:
    image: jonasal/kea-dhcp4:3.0.2
    container_name: nw-kea
    restart: unless-stopped
    network_mode: host
    command: -c /kea/config/dhcp4.json
    volumes:
      - ./config:/kea/config
      - ./sockets:/kea/sockets
      - ./leases:/kea/leases
      - ./logs:/kea/logs

  keastats:
    image: ghcr.io/netwarlan/kea-telegraf-plugin:latest
    container_name: nw-keastats
    volumes:
      - keastats-bin:/shared

  telegraf:
    image: telegraf:latest
    container_name: nw-kea-telegraf
    network_mode: "host"
    depends_on:
      keastats:
        condition: service_completed_successfully
    volumes:
      - ./telegraf.conf:/etc/telegraf/telegraf.conf:ro
      - keastats-bin:/opt/keastats:ro
    restart: unless-stopped

volumes:
  keastats-bin:
```

### telegraf.conf

```toml
[[outputs.influxdb_v2]]
  urls = ["http://<influxdb-host>:8181"]
  token = "<your-token>"
  organization = ""
  bucket = "kea"

[[inputs.exec]]
  commands = ["/opt/keastats/keastats --url http://localhost:8000/ --server dhcp-server-01"]
  timeout = "10s"
  interval = "15s"
  data_format = "influx"
```

## Versioning

This project uses automatic [semantic versioning](https://semver.org/) via [action-semantic-versioning](https://github.com/netwarlan/action-semantic-versioning). When commits are pushed to `main`, the action parses commit messages using [Conventional Commits](https://www.conventionalcommits.org/) and creates a git tag and GitHub release.

| Commit prefix | Bump | Example |
|---------------|------|---------|
| `fix:` | patch | `v1.0.0` → `v1.0.1` |
| `feat:` | minor | `v1.0.0` → `v1.1.0` |
| `BREAKING CHANGE:` or `!` | major | `v1.0.0` → `v2.0.0` |

Commits that don't match a conventional type (e.g. `docs:`, `chore:`) are skipped.

## Building

```bash
make build    # compile to bin/keastats
make test     # run tests with race detection
make lint     # run golangci-lint
make docker   # build Docker image
make clean    # remove build artifacts
```

## Updating

Pull the latest image and recreate the init container:

```bash
docker compose pull keastats
docker compose up -d
```

Telegraf will pick up the new binary on its next restart.
