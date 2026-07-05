TEST_PKGS = ./_tests/config/... ./_tests/domain/... ./_tests/factory/... ./_tests/adapters/...

.PHONY: test test-unit test-integration test-race test-v test-cover test-run thirdparties-up thirdparties-down build run migration queue script job

# Unit + integration. Brings up the third-party services (Postgres/Redis/RabbitMQ)
# via Docker, then runs everything.
test:
	./scripts/run-tests.sh --all

# Unit tests only — fast, no Docker required.
test-unit:
	./scripts/run-tests.sh --unit

# Integration tests only — brings up the third-party services via Docker.
test-integration:
	./scripts/run-tests.sh --integration

# Start / stop the third-party services without running any tests.
thirdparties-up:
	./docker/thirdparties/run-thirdparties-docker.sh

thirdparties-down:
	./docker/thirdparties/run-thirdparties-docker.sh --stop

# Race detector over the unit suite plus the in-package (white-box) tests. No
# Docker. This is the lane that guards the concurrency-sensitive wire parser and
# connection paths — run it in CI so data races and parser panics are caught.
test-race:
	go test -race $(TEST_PKGS) ./internal/...

# The -v / -cover / -run helpers target the unit suite for quick iteration.
test-v:
	go test $(TEST_PKGS) -v

test-cover:
	go test $(TEST_PKGS) -cover

test-run:
	go test $(TEST_PKGS) -v -run $(T)

build:
	go build -o bin/gameserver ./cmd/gameserver

run:
	go run ./cmd/gameserver

migration:
	@if [ -z "$(name)" ]; then echo "Usage: make migration name=<migration_name>"; exit 1; fi
	@timestamp=$$(date +%Y%m%d%H%M%S); \
	file="external/migrations/$${timestamp}_$(name).go"; \
	sed -e "s|TIMESTAMP_NAME|$${timestamp}_$(name)|g" \
		-e "s|TIMESTAMP|$${timestamp}|g" \
		-e "s|NAME|$(name)|g" \
		external/migrations/migration.go.tmpl > "$$file"; \
	echo "Created $$file"

queue:
	@if [ -z "$(topic)" ]; then echo "Usage: make queue topic=<topic.name>"; exit 1; fi
	@mkdir -p external/queues; \
	file="external/queues/$$(echo $(topic) | tr './' '__').go"; \
	sed -e "s|TOPIC|$(topic)|g" \
		external/queues/queue.go.tmpl > "$$file"; \
	echo "Created $$file"

script:
	@if [ -z "$(name)" ]; then echo "Usage: make script name=<script_name>"; exit 1; fi
	@mkdir -p external/scripts; \
	file="external/scripts/$(name).lua"; \
	sed -e "s|NAME|$(name)|g" \
		external/scripts/script.lua.tmpl > "$$file"; \
	echo "Created $$file"

job:
	@if [ -z "$(name)" ] || [ -z "$(interval)" ]; then echo "Usage: make job name=<job_name> interval=<seconds>"; exit 1; fi
	@mkdir -p external/scripts/jobs; \
	lua="external/scripts/jobs/$(name).lua"; \
	if [ -f "$$lua" ]; then echo "$$lua already exists"; else \
		printf -- '-- @job interval=%s\n-- jobs/%s — recurring job. TODO: implement.\n' "$(interval)" "$(name)" > "$$lua"; \
		echo "Created $$lua (discovered by its @job header — no Go)"; \
	fi
