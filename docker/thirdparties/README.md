# Third-party services for integration tests

The integration test suites (`//go:build integration`) run against **real**
Postgres, Redis, and RabbitMQ instances — no mocks. This directory brings those
up with Docker, following the same shape as Apache Doris' `docker/thirdparties`:
one compose file per service, selectable, driven by a single settings file.

## Layout

```
docker/thirdparties/
├── custom_settings.env        # ports/credentials — single source of truth
├── run-thirdparties-docker.sh # start/stop the services
└── docker-compose/
    ├── postgres/postgres.yaml
    ├── redis/redis.yaml
    └── rabbitmq/rabbitmq.yaml
```

## Usage

```bash
# start everything (waits until healthy)
./docker/thirdparties/run-thirdparties-docker.sh
# or a subset
./docker/thirdparties/run-thirdparties-docker.sh -c redis,rabbitmq
# stop + remove (with volumes)
./docker/thirdparties/run-thirdparties-docker.sh --stop
```

You normally don't call it directly — the test runner does:

```bash
make test              # unit + integration (starts services automatically)
make test-unit         # unit only, no Docker
make test-integration  # integration only (starts services)
make thirdparties-up   # just start the services
make thirdparties-down # stop them
```

## Configuration

Edit `custom_settings.env` to change ports/credentials (e.g. if `5432` is already
taken locally). `scripts/run-tests.sh` derives the `TEST_*` env vars the Go
suites read from these same values, so you only change them in one place.
Anything already exported in the environment (e.g. in CI) wins over the file.

Data is ephemeral: Postgres runs on a tmpfs and Redis persistence is disabled,
so every `up` starts clean.
