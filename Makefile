.PHONY: all build run dev clean test install deploy docker-build docker-run docker-deploy release

ifeq ($(OS),Windows_NT)
    EXT := .exe
    RUN_SCRIPT := run.ps1
    BUILD_SCRIPT := build.ps1
    DOCKER_BUILD := docker-build.ps1
    DOCKER_RUN := docker-run.ps1
    DOCKER_DEPLOY := docker-deploy.ps1
    RELEASE_SCRIPT := release-build.ps1
else
    EXT :=
    RUN_SCRIPT := run.sh
    BUILD_SCRIPT := build.sh
    DOCKER_BUILD := docker-build.sh
    DOCKER_RUN := docker-run.sh
    DOCKER_DEPLOY := docker-deploy.sh
    RELEASE_SCRIPT := release-build.sh
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
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File "scripts/$(RUN_SCRIPT)"; \
	else \
		./scripts/$(RUN_SCRIPT); \
	fi

dev: run

deploy:
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/deploy.ps1; \
	else \
		./scripts/deploy.sh; \
	fi

docker-build:
	./scripts/$(DOCKER_BUILD)

docker-run:
	./scripts/$(DOCKER_RUN)

docker-deploy:
	./scripts/$(DOCKER_DEPLOY)

release:
	./scripts/$(RELEASE_SCRIPT)

clean:
	rm -rf packages/server/bin packages/web/dist dist
	-rm -rf packages/web/node_modules

test:
	cd packages/server && go test ./...
