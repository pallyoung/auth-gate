.PHONY: all build run dev clean test install

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := windows
    EXT := .exe
    NODE_modules := node_modules
else
    DETECTED_OS := unix
    EXT :=
    NODE_modules := node_modules
endif

all: build

build:
	@if [ "$(DETECTED_OS)" = "windows" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/build.ps1; \
	else \
		./scripts/build.sh; \
	fi

install:
	@if [ "$(DETECTED_OS)" = "windows" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/install.ps1; \
	else \
		./scripts/install.sh; \
	fi

run:
	@if [ "$(DETECTED_OS)" = "windows" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/run.ps1; \
	else \
		./scripts/run.sh; \
	fi

dev:
	@if [ "$(DETECTED_OS)" = "windows" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/run.ps1; \
	else \
		./scripts/dev.sh; \
	fi

clean:
	rm -rf packages/server/bin packages/web/dist
	@if [ "$(DETECTED_OS)" != "windows" ]; then \
		rm -rf packages/web/node_modules; \
	fi

test:
	cd packages/server && go test ./...
