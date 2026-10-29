.PHONY: all build run test lint clean install

BINARY_NAME=ventox
BUILD_DIR=bin
MODULE=github.com/0xzahed/ventox

all: build

build:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) cmd/ventox/main.go
	@echo "✔ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

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
