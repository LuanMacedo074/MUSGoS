TEST_PKGS = ./_tests/config/... ./_tests/domain/... ./_tests/factory/... ./_tests/adapters/...

.PHONY: test test-unit test-integration test-v test-cover test-run thirdparties-up thirdparties-down build run migration queue script job

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
	@mkdir -p external/jobs external/scripts/jobs; \
	file="external/jobs/$(name).go"; \
	sed -e "s|NAME|$(name)|g" -e "s|INTERVAL|$(interval)|g" \
		external/jobs/job.go.tmpl > "$$file"; \
	lua="external/scripts/jobs/$(name).lua"; \
	[ -f "$$lua" ] || printf -- '-- jobs/%s — recurring job (interval: %ss). TODO: implement.\n' "$(name)" "$(interval)" > "$$lua"; \
	echo "Created $$file and $$lua"
