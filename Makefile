# simpler2sync Makefile
# Requires: Go 1.22+, GCC (MinGW on Windows, build-essential on Linux, Xcode CLI on macOS)

BIN_DIR   := bin
APP_NAME  := simpler2sync
LDFLAGS   := -s -w
GO        := go

.PHONY: help run build build-all package package-all lint clean deps

## help: Show all available targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed 's/^/  /'

## run: Build and run locally
run:
	CGO_ENABLED=1 $(GO) run .

## build: Build binary for current platform
build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) .

## build-all: Cross-compile for Windows, macOS, Linux
build-all: build-windows build-macos build-linux

build-windows:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-windows-amd64.exe .

build-macos:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-amd64 .
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 .

build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 .

## package: Package current platform binary into .zip
package: build
	@mkdir -p $(BIN_DIR)/dist
	@cp $(BIN_DIR)/$(APP_NAME)* $(BIN_DIR)/dist/
	@cp README.md $(BIN_DIR)/dist/ 2>/dev/null || true
	@cd $(BIN_DIR)/dist && zip -r ../$(APP_NAME)-package.zip .
	@echo "Package: $(BIN_DIR)/$(APP_NAME)-package.zip"

## package-all: Cross-compile and package all platforms
package-all: build-all
	@rm -rf $(BIN_DIR)/dist
	@mkdir -p $(BIN_DIR)/dist
	@cp README.md $(BIN_DIR)/dist/ 2>/dev/null || true
	@for BIN in $(BIN_DIR)/$(APP_NAME)-*; do \
		OS=$$(echo $$BIN | sed 's/.*-\([a-z]*\)-.*/\1/'); \
		ARCH=$$(echo $$BIN | sed 's/.*-\(amd64\|arm64\).*/\1/'); \
		DIR=$(BIN_DIR)/dist/$(APP_NAME)-$$OS-$$ARCH; \
		mkdir -p $$DIR; \
		cp $$BIN $$DIR/; \
		cp README.md $$DIR/ 2>/dev/null || true; \
		(cd $(BIN_DIR)/dist && zip -r ../$(APP_NAME)-$$OS-$$ARCH.zip $(APP_NAME)-$$OS-$$ARCH/*); \
	done
	@rm -rf $(BIN_DIR)/dist
	@echo "Packages in $(BIN_DIR)/"

## lint: Run go vet and static analysis
lint:
	$(GO) vet ./internal/config/ ./internal/store/ ./internal/r2client/ ./internal/sync/ ./internal/scheduler/
	@echo "(skipping gui/ - requires CGo/OpenGL)"

## deps: Download and tidy dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR)

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...
