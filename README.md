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
make test               # unit + integration (starts Docker third-party services)
make test-unit          # unit tests only (fast, no Docker)
make test-integration   # integration tests only (starts Docker third-party services)
make test-v             # unit tests with verbose output
make test-cover         # unit tests with a coverage report
make test-run T=Name    # run a specific unit test by name
make thirdparties-up    # start the third-party services (Postgres/Redis/RabbitMQ)
make thirdparties-down  # stop them
make build              # build to bin/gameserver
make run                # run the server
```

Integration tests (build tag `integration`) run against **real** Postgres, Redis,
and RabbitMQ brought up via Docker — see [`docker/thirdparties/`](docker/thirdparties/).
If a default port is already in use locally, override it in
`docker/thirdparties/custom_settings.env`. Unit tests need no Docker.

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
| `SANDBOXED_SCRIPTS_DIR` | _(empty)_ | Directory under `SCRIPTS_PATH` that `mus.loadSandboxed(id)` reads; empty disables it |
| `JOBS_ENABLED` | `1` | Enable scheduled jobs |
| `DISCONNECT_HOOK` | _(empty)_ | Script subject invoked when a client disconnects; empty disables it |
| `DEFAULT_MOVIE_ID` | _(empty)_ | Movie that `@group` sends from movie-less senders (system scripts, jobs) resolve against; empty disables the fallback |
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

Detailed docs in [`docs/architecture.md`](docs/architecture.md); the auth
stores behind `mus.token` and `mus.refresh` are in
[`docs/refresh-tokens.md`](docs/refresh-tokens.md).

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

## Syncing the engine with FsosServer

[FsosServer](https://github.com/LuanMacedo074/FSOSServer) is a game built on a
fork of this engine. The two repos share history and the same engine tree:

- **Engine** (kept identical in both repos): `internal/`, `cmd/gameserver/`, and
  the engine tests in `_tests/{adapters,config,domain,factory,testutil}`.
- **Application** (never synced): `external/` and, in FsosServer, the game
  tests in `_tests/game/`.

Only the Go module path differs: `musgos` here, `fsos-server` in FsosServer.
That makes a plain `git cherry-pick` fail on every import line, so engine
commits cross between the repos as patches with the import path rewritten.

Engine changes should land in MUSGoS first, but they may land in either repo.
To carry commits from FsosServer into MUSGoS (run from this repo, with
FsosServer checked out next to it):

```bash
git -C ../FsosServer format-patch --stdout <base>..<tip> -- \
    internal cmd/gameserver _tests/adapters _tests/config _tests/domain _tests/factory _tests/testutil \
  | sed 's#"fsos-server/#"musgos/#g' \
  | git am -3
gofmt -w internal cmd _tests   # the rewrite can reorder an import group
```

To carry commits the other way, run it from FsosServer with the repos swapped
and `sed 's#"musgos/#"fsos-server/#g'`. The pathspec drops the application
half of a mixed commit. If `git am` stops on a conflict, resolve it, then
`git am --continue`.

After a sync, check that the engine trees still match:

```bash
scripts/engine-drift.sh ../FsosServer          # file list; exit 1 on drift
scripts/engine-drift.sh ../FsosServer --full   # unified diff
```

The script rewrites the module path and runs gofmt on both trees before it
diffs them, so it reports only real drift. `internal/` and `cmd/gameserver/`
should always match. Today the engine tests still drift: some tests were
renamed or improved in one repo and not yet carried to the other.

## Credits

The Blowfish implementation is based on [OpenSMUS](https://github.com/piacentini/OpenSMUS)
by Mauricio Piacentini, licensed under MIT.
