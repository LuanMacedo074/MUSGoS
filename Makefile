TEST_PKGS = ./_tests/config/... ./_tests/domain/... ./_tests/factory/... ./_tests/adapters/...

.PHONY: test test-v test-cover test-run build run migration

test:
	go test $(TEST_PKGS)

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
	sed -e "s/TIMESTAMP_NAME/$${timestamp}_$(name)/g" \
		-e "s/TIMESTAMP/$${timestamp}/g" \
		-e "s/NAME/$(name)/g" \
		external/migrations/migration.go.tmpl > "$$file"; \
	echo "Created $$file"
