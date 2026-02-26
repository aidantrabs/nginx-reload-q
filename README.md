# nginx-reload-q

A lightweight Go daemon that serializes nginx reload requests through a Unix domain socket. Built for environments where multiple processes (CI pipelines, deploy scripts, config management tools) might trigger nginx reloads concurrently.

## the problem

If you run nginx as a reverse proxy in front of multiple services, each service's deploy pipeline probably needs to reload nginx after updating its config. During a busy deploy window, this happens:

```
deploy-service-a: nginx -s reload  ─┐
deploy-service-b: nginx -s reload  ─┤  all at the same time
deploy-service-c: nginx -s reload  ─┤
deploy-service-d: nginx -s reload  ─┘
```

This causes real issues:

- **Race conditions** - config files are being written while another reload is already testing them with `nginx -t`. The test passes on a half-written config, or fails on a config that would be valid 200ms later.
- **Wasted work** - 10 deploys finishing at once means 10 identical reloads. nginx reload is idempotent, one reload picks up all config changes. The other 9 are pointless.
- **Unpredictable failures** - reloads can fail intermittently when they overlap, leading to confusing CI failures that pass on retry. The kind of bug that wastes hours.
- **No visibility** - when something does go wrong, there's no centralized log of what triggered a reload and when. You're grepping through 10 different pipeline logs.

## how it works

Instead of calling `nginx -s reload` directly, your deploy scripts send a `RELOAD` command to a Unix socket:

```
deploy-service-a: echo "RELOAD" > sock  ─┐
deploy-service-b: echo "RELOAD" > sock  ─┤
deploy-service-c: echo "RELOAD" > sock  ─┤  nginx-reload-q
deploy-service-d: echo "RELOAD" > sock  ─┘  serializes these
                                              into 1-2 actual
                                              nginx reloads
```

The daemon:
1. Accepts reload requests over a Unix domain socket
2. Deduplicates bursts - if a reload is already pending, new requests collapse into it
3. Runs `nginx -t` before every reload to catch config errors
4. Executes `nginx -s reload` only if the config test passes
5. Returns `OK` or `ERROR` to each client so your CI knows what happened
6. Logs everything as structured JSON

Only one reload ever runs at a time. The queue guarantees serialization.

## install

```bash
go install github.com/aidantrabs/nginx-reload-q/cmd/server@latest
```

Or build from source:

```bash
git clone https://github.com/aidantrabs/nginx-reload-q.git
cd nginx-reload-q
make build
```

Or with Docker:

```bash
docker build -t nginx-reload-q .
docker run -v /var/run:/var/run nginx-reload-q
```

## usage

### start the daemon

```bash
sudo ./nginx-reload-q
```

By default it listens on `/var/run/nginx-reload.sock` and exposes metrics on `127.0.0.1:9111`. Both are configurable via flags or env vars:

```bash
# flags
sudo ./nginx-reload-q -socket /tmp/reload.sock -metrics 0.0.0.0:9111

# env vars
RELOAD_SOCKET=/tmp/reload.sock RELOAD_METRICS_ADDR=0.0.0.0:9111 sudo ./nginx-reload-q
```

| setting | flag | env var | default |
|---------|------|---------|---------|
| socket path | `-socket` | `RELOAD_SOCKET` | `/var/run/nginx-reload.sock` |
| metrics address | `-metrics` | `RELOAD_METRICS_ADDR` | `127.0.0.1:9111` |

Flags take priority over env vars.

### send a reload request

From a deploy script or CI pipeline:

```bash
echo "RELOAD" | socat - UNIX-CONNECT:/var/run/nginx-reload.sock
```

Or with netcat:

```bash
echo "RELOAD" | nc -U /var/run/nginx-reload.sock
```

The response is either `OK` (reload was queued/deduplicated) or `ERROR: <reason>`.

### check metrics

```bash
curl http://127.0.0.1:9111/metrics
```

```json
{
  "reloads": 42,
  "failures": 1,
  "last_reload": "2026-02-26T12:30:00Z",
  "pending": false,
  "deduplicated": 87
}
```

### github actions example

```yaml
- name: reload nginx
  run: |
    response=$(echo "RELOAD" | socat - UNIX-CONNECT:/var/run/nginx-reload.sock)
    if [ "$response" != "OK" ]; then
      echo "nginx reload failed: $response"
      exit 1
    fi
```

### systemd

```ini
[Unit]
Description=nginx reload queue
After=nginx.service
Requires=nginx.service

[Service]
Type=simple
ExecStart=/usr/local/bin/nginx-reload-q
Environment=RELOAD_SOCKET=/var/run/nginx-reload.sock
Environment=RELOAD_METRICS_ADDR=127.0.0.1:9111
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## testing

```bash
go test ./...
```

## protocol

Dead simple text protocol over Unix socket. One command per connection.

```
Client sends:    RELOAD\n
Server responds: OK\n
                 ERROR: <message>\n
```

That's it. No framing, no headers, no JSON. Easy to use from any language or shell script.

## architecture

```
client ──> unix socket ──> queue ──> worker ──> nginx -t && nginx -s reload
                             │
                             └── dedup: collapses bursts into single reload
```

- `cmd/server/` - entrypoint
- `internal/socket/` - unix domain socket server
- `internal/queue/` - single-worker queue with burst deduplication
- `internal/reloader/` - runs nginx -t and nginx -s reload
- `internal/metrics/` - JSON metrics endpoint
- `internal/logging/` - structured JSON logging

## license

MIT
