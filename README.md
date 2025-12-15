# immich-kiosk-scheduler

A lightweight scheduling proxy for [Immich Kiosk](https://github.com/damongolding/immich-kiosk) that automatically rotates photo albums based on a date-based schedule.

Perfect for digital photo frames that should display seasonal content - Christmas photos in December, summer vacation photos in July, etc.

## Features

- **Date-based album scheduling** - Define date ranges for each album
- **Year-wrap support** - Schedules can cross year boundaries (e.g., Nov 15 to Jan 1)
- **First-match wins** - Overlapping schedules are handled predictably
- **Passthrough parameters** - Forward Immich Kiosk settings (transition, duration, etc.)
- **Prometheus metrics** - Monitor redirects and current schedule
- **Health endpoint** - Kubernetes-ready health checks
- **Minimal footprint** - Single static binary, runs from scratch container

## Quick Start

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  ghcr.io/sharkusmanch/immich-kiosk-scheduler:latest \
  serve --config /config.yaml
```

### Binary

```bash
# Download the latest release
curl -LO https://github.com/sharkusmanch/immich-kiosk-scheduler/releases/latest/download/immich-kiosk-scheduler_linux_amd64

# Make executable
chmod +x immich-kiosk-scheduler_linux_amd64

# Run
./immich-kiosk-scheduler_linux_amd64 serve --config config.yaml
```

## Configuration

Create a `config.yaml` file:

```yaml
# Base URL of your Immich Kiosk instance
kiosk_url: "https://kiosk.example.com"

# Default album when no schedule matches
default_album: "your-default-album-uuid"

# Port to listen on
port: 8080

# Parameters to pass through to Immich Kiosk
passthrough_params:
  - transition
  - duration

# Date-based schedule (first match wins)
schedule:
  - name: christmas
    album: "christmas-album-uuid"
    start: "11-15"  # Nov 15
    end: "01-01"    # Jan 1

  - name: summer
    album: "summer-album-uuid"
    start: "06-21"
    end: "09-21"
```

### Configuration Options

| Option | Description | Default | Env Var |
|--------|-------------|---------|---------|
| `kiosk_url` | Immich Kiosk base URL | *required* | `IKS_KIOSK_URL` |
| `default_album` | Album ID when no schedule matches | *required* | `IKS_DEFAULT_ALBUM` |
| `port` | HTTP server port | `8080` | `IKS_PORT` |
| `log_level` | Logging level (debug/info/warn/error) | `info` | `IKS_LOG_LEVEL` |
| `passthrough_params` | Query params to forward | `[]` | - |
| `schedule` | List of schedule entries | `[]` | - |

### Schedule Entry

| Field | Description | Format |
|-------|-------------|--------|
| `name` | Human-readable name | string |
| `album` | Immich album UUID | string |
| `start` | Start date (inclusive) | `MM-DD` |
| `end` | End date (inclusive) | `MM-DD` |

### Environment Variables

Non-schedule configuration can be set via environment variables with the `IKS_` prefix:

```bash
export IKS_CONFIG=/path/to/config.yaml
export IKS_KIOSK_URL=https://kiosk.example.com
export IKS_DEFAULT_ALBUM=abc-123
export IKS_PORT=3000
export IKS_LOG_LEVEL=debug
```

### CLI Flags

```bash
# Global flags
--config string      Config file path (default: ./config.yaml)
--log-level string   Log level (default: info)

# Serve command
--port int           Port to listen on (default: 8080)

# Test command
--date string        Date to test (MM-DD format, defaults to today)
```

## Usage

### Running the Server

```bash
# Using config file
immich-kiosk-scheduler serve --config config.yaml

# With port override
immich-kiosk-scheduler serve --config config.yaml --port 3000

# Using environment variables
IKS_CONFIG=/etc/iks/config.yaml immich-kiosk-scheduler serve
```

### Testing the Schedule

Verify which album would be selected for a specific date:

```bash
# Test today's date
immich-kiosk-scheduler test --config config.yaml

# Test a specific date
immich-kiosk-scheduler test --config config.yaml --date 12-25
```

Example output:
```
Testing schedule for December 25

Schedule:  christmas
Album ID:  d2459437-3267-47ea-a421-9bfeedde604d
Redirect:  https://kiosk.example.com?album=d2459437-3267-47ea-a421-9bfeedde604d
```

## Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Redirect to Immich Kiosk with scheduled album |
| `GET /healthz` | Health check (returns JSON with status and current schedule) |
| `GET /metrics` | Prometheus metrics |

## Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `immich_kiosk_scheduler_redirects_total` | Counter | Total redirects by schedule name |
| `immich_kiosk_scheduler_current_schedule` | Gauge | Currently active schedule (1 = active) |

## Integration

### With Fully Kiosk Browser

Set the Start URL to your immich-kiosk-scheduler instance:

```
http://immich-kiosk-scheduler:8080/
```

The scheduler will redirect to Immich Kiosk with the appropriate album.

### With Home Assistant

Point your iframe card to the scheduler:

```yaml
type: iframe
url: http://immich-kiosk-scheduler:8080/
```

### Kubernetes / Helm

See the [deployment example](deploy/kubernetes/) for a complete Kubernetes deployment.

## Building from Source

```bash
# Clone the repository
git clone https://github.com/sharkusmanch/immich-kiosk-scheduler.git
cd immich-kiosk-scheduler

# Build
go build -o immich-kiosk-scheduler ./cmd/immich-kiosk-scheduler

# Run tests
go test ./...

# Build Docker image
docker build -t immich-kiosk-scheduler .
```

## Finding Album IDs

1. Open Immich web UI
2. Navigate to the album you want to use
3. Copy the UUID from the URL:
   ```
   https://immich.example.com/albums/d2459437-3267-47ea-a421-9bfeedde604d
                                      ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                      This is your album ID
   ```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Immich](https://immich.app/) - Self-hosted photo management
- [Immich Kiosk](https://github.com/damongolding/immich-kiosk) - Slideshow display for Immich
