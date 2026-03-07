.PHONY: all build build-arm64 clean deploy symlink test lint

BINARY_NAME=unas-custom
BUILD_DIR=build
CMD_DIR=cmd/unas-custom

all: build-arm64 deploy

build:
	@echo "Building for local platform..."
	GOOS=linux go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

build-arm64:
	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 ./$(CMD_DIR)
	@echo "Binary size: $$(ls -lh $(BUILD_DIR)/$(BINARY_NAME)-arm64 | awk '{print $$5}')"

clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)

symlink: deploy
	@echo "Creating symlinks..."
	ln -sf $(BINARY_NAME) $(BUILD_DIR)/deploy/exportfs
	ln -sf $(BINARY_NAME) $(BUILD_DIR)/deploy/smbcontrol
	@echo "Symlinks created: exportfs -> $(BINARY_NAME), smbcontrol -> $(BINARY_NAME)"

test:
	@echo "Running tests..."
	go test -v -race ./...

lint:
	@echo "Linting code..."
	go vet ./... && test -z "$$(gofmt -l .)"
	@echo "Lint passed!"

deploy: build-arm64
	@echo "Creating deployment package..."
	mkdir -p $(BUILD_DIR)/deploy
	cp $(BUILD_DIR)/$(BINARY_NAME)-arm64 $(BUILD_DIR)/deploy/$(BINARY_NAME)
	cp scripts/* $(BUILD_DIR)/deploy/
	cp config.example.yaml $(BUILD_DIR)/deploy/
	chmod +x $(BUILD_DIR)/deploy/$(BINARY_NAME)
	chmod +x $(BUILD_DIR)/deploy/*.sh
	@echo ""
	@echo "Deployment package ready in $(BUILD_DIR)/deploy/"
	@echo "Copy to UNAS: scp -r $(BUILD_DIR)/deploy/* root@unas:/persistent/unas-custom/"
