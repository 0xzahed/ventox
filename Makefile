.PHONY: all build build-darwin build-all run test lint clean install

BINARY_NAME=ventox
BUILD_DIR=bin
MODULE=github.com/0xzahed/ventox

all: build

build:
	@echo "🔨 Building $(BINARY_NAME) for current OS..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/ventox/main.go
	@echo "✔ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-darwin:
	@echo "🍏 Building for macOS (Darwin arm64 & amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/ventox/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/ventox/main.go
	@echo "✔ macOS builds complete in $(BUILD_DIR)/"

build-all: build build-darwin
	@echo "🐧🍏 All platform builds complete!"

run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "🧪 Running unit tests..."
	go test -v -race ./...

lint:
	@echo "🔍 Running code checks..."
	go vet ./...

install: build
	@echo "📦 Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✔ Installed! You can now run '$(BINARY_NAME)' from anywhere."

uninstall:
	@echo "🗑️ Removing /usr/local/bin/$(BINARY_NAME)..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✔ Uninstalled."

clean:
	@rm -rf $(BUILD_DIR)
	@echo "🧹 Cleaned build artifacts."
