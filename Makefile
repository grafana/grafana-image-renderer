OUT_DIR = dist
BINARY ?= $(OUT_DIR)/grafana-image-renderer

.PHONY: check
check: lint test

.PHONY: test
test:
	go test ./... -timeout=60s

.PHONY: test-acceptance
test-acceptance:
	docker build -t gir .
	IMAGE=gir go test ./tests/acceptance/... -timeout=60s

.PHONY: lint
lint:
	go tool goimports -l .
	golangci-lint run

.PHONY: fix
fix:
	go tool goimports -w .
	golangci-lint run --fix

.PHONY: build
build: $(OUT_DIR)
	go build -buildvcs -o $(BINARY) .

.PHONY: build-all
build-all: build $(OUT_DIR)
	GOOS=linux GOARCH=amd64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build -buildvcs -o $(OUT_DIR)/grafana-image-renderer-windows-arm64.exe .

.PHONY: clean
clean:
	rm -rf "$(OUT_DIR)"

.PHONY: docs-dev
docs-dev:
	make -C docs docs

.PHONY: docs
docs:
	make -C docs update vale

.PHONY: all
all: lint build-all test test-acceptance

$(OUT_DIR):
	mkdir -p "$(OUT_DIR)"
