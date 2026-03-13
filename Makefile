TEST_PKGS = ./_tests/config/... ./_tests/domain/... ./_tests/factory/... ./_tests/adapters/...

.PHONY: test test-v test-cover test-run build run

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
