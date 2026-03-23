APP_NAME=timecard
BUILD_DIR=bin
GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Default values (can be overridden)
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all build build-all clean test

# Usage:
#   make build           				# builds for your current system, output: bin/timecard
#   make build-all      				# builds for all supported systems, output: bin/timecard-<os>-<arch>-<hash>
#   GOOS=linux GOARCH=amd64 make build  # cross-compiles for linux/amd64

all: build-all

build:
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BUILD_DIR)/$(APP_NAME) ./main.go
	@echo "Built $(APP_NAME) successfully to target /$(BUILD_DIR)"

build-all:
	@echo "Building $(APP_NAME) binaries for all supported OS/ARCH combinations..."
	@rm -rf $(BUILD_DIR)/$(APP_NAME)*
	@mkdir -p $(BUILD_DIR)
	@GIT_HASH=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'main.BuildGitHash=$$GIT_HASH' -X 'main.BuildLatestHash=$$GIT_HASH'" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64-$$GIT_HASH ./main.go; \
	GOOS=linux GOARCH=arm64 go build -ldflags "-X 'main.BuildGitHash=$$GIT_HASH' -X 'main.BuildLatestHash=$$GIT_HASH'" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64-$$GIT_HASH ./main.go; \
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'main.BuildGitHash=$$GIT_HASH' -X 'main.BuildLatestHash=$$GIT_HASH'" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64-$$GIT_HASH ./main.go; \
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X 'main.BuildGitHash=$$GIT_HASH' -X 'main.BuildLatestHash=$$GIT_HASH'" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64-$$GIT_HASH ./main.go
	@echo "Built $(APP_NAME) successfully to target /$(BUILD_DIR)"

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./...
