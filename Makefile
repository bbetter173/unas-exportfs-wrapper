.PHONY: all build build-arm64 clean deploy

BINARY_NAME=exportfs-wrapper
BUILD_DIR=build
CMD_DIR=cmd/exportfs-wrapper

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
	@echo "Copy to UNAS: scp -r $(BUILD_DIR)/deploy/* root@unas:/persistent/nfs-intercept/"
