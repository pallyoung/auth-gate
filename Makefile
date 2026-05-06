.PHONY: all build run dev clean test install deploy

ifeq ($(OS),Windows_NT)
    EXT := .exe
    RUN_SCRIPT := run.ps1
else
    EXT :=
    RUN_SCRIPT := run.sh
endif

all: build

build:
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/build.ps1; \
	else \
		./scripts/build.sh; \
	fi

install:
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/install.ps1; \
	else \
		./scripts/install.sh; \
	fi

run:
	@powershell -ExecutionPolicy Bypass -File "scripts/$(RUN_SCRIPT)"

dev: run

deploy:
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/deploy.ps1; \
	else \
		./scripts/deploy.sh; \
	fi

clean:
	rm -rf packages/server/bin packages/web/dist
	-rm -rf packages/web/node_modules

test:
	cd packages/server && go test ./...
