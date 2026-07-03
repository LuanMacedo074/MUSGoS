# MUSGoS

MUS (Multiuser Server) written in Go — a modern, hexagonal reimplementation of the
Macromedia Shockwave Multiuser Server, compatible with Shockwave/Director clients
that speak the SMUS protocol.

## Requirements

- Go 1.24+

## Quick start

```bash
cp .env.example .env    # configure the variables
make run                # start the server
```

## Commands

```bash
make test               # run all tests
make test-v             # tests with verbose output
make test-cover         # tests with a coverage report
make test-run T=Name    # run a specific test by name
make build              # build to bin/gameserver
make run                # run the server
```

## Configuration

Via environment variables. Core settings below; **see `.env.example` for the full
set** (Redis, RabbitMQ, cache, rate limiting, metrics, UDP, and more).

| Variable | Default | Description |
|---|---|---|
| `APPLICATION_NAME` | `SMUS-SERVER` | Application name |
| `PORT` | `1199` | Server TCP port |
| `ENVIRONMENT` | `development` | Runtime environment |
| `MAX_MESSAGE_SIZE` | `2097151` | Max message size (bytes) |
| `DEFAULT_USER_LEVEL` | `20` | Default user level on logon |
| `LOG_LEVEL` | `DEBUG` | Log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `LOGGER_TYPE` | `file` | Logger type |
| `CIPHER_TYPE` | `blowfish` | Cipher type |
| `ENCRYPTION_KEY` | `IPAddress resolution` | Encryption key (a `#All` prefix encrypts whole packets) |
| `PROTOCOL` | `smus` | Communication protocol |
| `DATABASE_TYPE` | `sqlite` | Database type (`sqlite`, `postgres`) |
| `DATABASE_PATH` | `data/musgo.db` | Database file path (sqlite) |
| `DATABASE_URL` | — | Full Postgres DSN; overrides the discrete `DATABASE_*` fields below |
| `DATABASE_HOST` | `localhost` | Postgres host |
| `DATABASE_PORT` | `5432` | Postgres port |
| `DATABASE_USER` | `postgres` | Postgres user |
| `DATABASE_PASSWORD` | — | Postgres password |
| `DATABASE_NAME` | `musgo` | Postgres database name |
| `DATABASE_SSLMODE` | `disable` | Postgres SSL mode |
| `SCRIPTS_PATH` | `external/scripts` | Lua scripts path |
| `SCRIPT_TIMEOUT` | `5` | Lua script timeout (seconds) |
| `JOBS_ENABLED` | `1` | Enable scheduled jobs |
| `DISCONNECT_HOOK` | `users/onDisconnect` | Script subject invoked when a client disconnects |
| `AUTH_MODE` | `open` | Auth mode (`none`, `open`, `strict`) |
| `SESSION_STORE_TYPE` | `memory` | Session store (`memory`, `redis`) |
| `QUEUE_TYPE` | `memory` | Message queue (`memory`, `redis`, `rabbitmq`) |
| `CACHE_TYPE` | `memory` | Cache (`memory`, `redis`) |

## Architecture

The project uses **hexagonal architecture** (ports & adapters). The domain defines
interfaces (ports), and the concrete implementations (adapters) are injected via
factories.

The ports (`internal/domain/ports/`): `Cipher`, `Handler`, `Logger`, `Database`,
`QueryBuilder`, `SessionStore`, `MessageSender`, `ConnectionWriter`, `Migration`,
`Schema`, `Queue`, `ScriptEngine`, `Cache`, `RateLimiter`, `Metrics`, `Timer`, `Email`.

```
internal/
├── config/              ← environment variables
├── factory/             ← resolves concrete implementations
├── domain/
│   ├── types/
│   │   ├── lingo/       ← Lingo types (LValue, LString, LInteger, etc.)
│   │   └── smus/        ← SMUS protocol (MUSMessage, headers)
│   └── ports/           ← interfaces (Cipher, Handler, Logger, Database, Queue, …)
└── adapters/
    ├── inbound/         ← TCP server, SMUS handler
    └── outbound/        ← Blowfish cipher, SQLite, loggers, queues, …
```

Dependency rule: import arrows always point toward the domain. Adapters depend on
the domain, never the other way around.

Detailed docs in [`docs/architecture.md`](docs/architecture.md).

## Tests

Tests live in `_tests/` (an underscore directory, ignored by `go test ./...`). Run
them with `make test`.

```
_tests/
├── testutil/            ← shared mocks
├── config/              ← configuration tests
├── domain/              ← type and port tests
├── factory/             ← factory tests
└── adapters/            ← adapter tests
```

## Credits

The Blowfish implementation is based on [OpenSMUS](https://github.com/piacentini/OpenSMUS)
by Mauricio Piacentini, licensed under MIT.
