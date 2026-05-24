# Redis — Go Implementation

A from-scratch Redis server and interactive client CLI written in Go.

---

## What's in this repo

| Binary | Path | Description |
|---|---|---|
| `redis-server` | `app/` | Full Redis server — RESP protocol, persistence, replication, pub/sub |
| `redis-cli` | `cmd/redis-cli/` | Rich interactive client — TUI REPL, autocomplete, live pub/sub feed |

---

## Server

### Run

```sh
# Default (port 6379)
go run ./app/

# Custom port
go run ./app/ --port 6380

# With RDB persistence
go run ./app/ --port 6379 --dir /tmp --dbfilename dump.rdb

# With AOF persistence
go run ./app/ --port 6379 --appendonly yes --appendfilename appendonly.aof

# As a replica
go run ./app/ --port 6380 --replicaof "127.0.0.1 6379"
```

### Supported commands

| Category | Commands |
|---|---|
| Basic | `PING` `ECHO` `TYPE` `INFO` |
| String | `SET` `GET` `INCR` `DECR` |
| List | `LPUSH` `RPUSH` `LPOP` `RPOP` `LLEN` `LRANGE` `BLPOP` |
| Stream | `XADD` `XRANGE` `XREAD` |
| Sorted set | `ZADD` `ZRANGE` `ZRANK` `ZSCORE` `ZCARD` `ZREM` |
| Geo | `GEOADD` `GEOPOS` `GEODIST` `GEOSEARCH` |
| Pub/Sub | `SUBSCRIBE` `PSUBSCRIBE` `UNSUBSCRIBE` `PUNSUBSCRIBE` `PUBLISH` |
| Transaction | `MULTI` `EXEC` `DISCARD` `WATCH` `UNWATCH` |
| Replication | `REPLCONF` `PSYNC` `WAIT` |
| ACL / Auth | `AUTH` `ACL` |
| Config / Keys | `CONFIG` `KEYS` |

### Architecture

```
app/
└── main.go                   Entry point, flag parsing

internal/
├── server/
│   ├── server.go             TCP listener, one goroutine per connection
│   ├── replication.go        Master/replica handshake and propagation
│   └── replicas.go           Replica tracking and WAIT logic
├── handler/                  One file per command group
├── store/                    In-memory data store (RWMutex protected)
│   ├── store.go              String and expiry management
│   ├── list.go               List operations
│   ├── zset.go               Sorted set (skip list backed)
│   ├── stream.go             Stream entries
│   └── geo.go                Geo indexing (Haversine distance)
├── resp/                     RESP encoder / decoder
├── rdb/                      RDB file reader for persistence
├── aof/                      Append-only file writer
├── pubsub/                   Pub/Sub broker (channel fan-out)
├── acl/                      User management and AUTH
└── command/                  Command name constants
```

**Concurrency model:** each client connection runs in its own goroutine; the store is protected by `sync.RWMutex` so reads run concurrently and writes are serialised — similar to Redis but via OS threads instead of an event loop.

---

## CLI

### Run

```sh
# Connect to local server
go run ./cmd/redis-cli/

# Custom host / port / password
go run ./cmd/redis-cli/ --host 127.0.0.1 --port 6380 --password secret

# Scripted demo (good for recording)
go run ./cmd/redis-cli/ --demo
```

### Features

- **Startup banner** — ASCII logo, connection status, ping latency, server version
- **Autocomplete** — fuzzy command suggestions with synopsis, context-aware (suppressed during pub/sub)
- **Typo correction** — Levenshtein distance hint when a command isn't recognised
- **Color-coded output** — ✓ green for OK, ✗ red for errors, cyan integers, dim nil
- **Sorted set tables** — `ZRANGE ... WITHSCORES` renders as an aligned member/score table
- **Stream cards** — `XRANGE`/`XREAD` renders each entry with ID and field/value rows
- **GEO tables** — `GEOPOS`/`GEODIST` rendered as labelled coordinate tables
- **Transaction mode** — prompt changes to `TX›` (yellow), queued commands show `⬡ QUEUED`
- **Pub/Sub live feed** — Bubble Tea view with timestamped `🔔` messages, `Ctrl+C` to exit and unsubscribe
- **Built-in help** — `HELP` lists all commands by category; `HELP <cmd>` shows synopsis and example

### Architecture

```
cmd/redis-cli/
└── main.go               Cobra entry point (--host, --port, --password, --demo)

internal/cli/
├── client/               TCP connection, RESP parser, latency tracking
├── state/                Shared session state (TX queue, subscriptions, latency)
├── registry/             Metadata for all 45+ commands (synopsis, args, examples)
├── completer/            go-prompt autocomplete + Levenshtein typo correction
├── renderer/
│   ├── renderer.go       RESP dispatcher
│   ├── string.go         Simple strings, errors, integers, bulk strings, null
│   ├── array.go          Generic arrays and EXEC indexed results
│   ├── table.go          Sorted set and GEO tables
│   ├── stream.go         XRANGE / XREAD stream entry cards
│   └── pubsub.go         Bubble Tea live-feed model
├── repl/                 go-prompt REPL loop and state machine
├── banner/               Startup banner printer
└── demo/                 Scripted demo runner (--demo flag)
```

---

## Build

```sh
# Build both binaries
go build -o redis-server ./app/
go build -o redis-cli    ./cmd/redis-cli/

# Run all tests
go test ./...
```

## Requirements

- Go 1.22+
