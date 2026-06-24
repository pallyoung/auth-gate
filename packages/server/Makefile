.PHONY: all build run dev clean web-deps web-build

# Build paths
BINARY=auth-gate
BUILD_DIR=bin
WEB_DIR=../web

all: build

build: web-build
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/server

run: build
	$(BUILD_DIR)/$(BINARY) start

dev:
	DEBUG=true air

web-deps:
	cd $(WEB_DIR) && npm install

web-build: web-deps
	cd $(WEB_DIR) && npm run build

clean:
	rm -rf $(BUILD_DIR) $(WEB_DIR)/dist $(WEB_DIR)/node_modules data
