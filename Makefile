.PHONY: all build run dev clean test install deploy docker-build docker-run docker-deploy

ifeq ($(OS),Windows_NT)
    DOCKER_BUILD := scripts/docker-build.ps1
    DOCKER_RUN := scripts/docker-run.ps1
    DOCKER_DEPLOY := scripts/docker-deploy.ps1
else
    DOCKER_BUILD := scripts/docker-build.sh
    DOCKER_RUN := scripts/docker-run.sh
    DOCKER_DEPLOY := scripts/docker-deploy.sh
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
	@powershell -ExecutionPolicy Bypass -File scripts/run.ps1

dev: run

deploy:
	@if [ "$(OS)" = "Windows_NT" ]; then \
		powershell -ExecutionPolicy Bypass -File scripts/deploy.ps1; \
	else \
		./scripts/deploy.sh; \
	fi

docker-build:
	@powershell -ExecutionPolicy Bypass -File "$(DOCKER_BUILD)"

docker-run:
	@powershell -ExecutionPolicy Bypass -File "$(DOCKER_RUN)"

docker-deploy:
	@powershell -ExecutionPolicy Bypass -File "$(DOCKER_DEPLOY)"

clean:
	rm -rf packages/server/bin packages/web/dist
	-rm -rf packages/web/node_modules

test:
	cd packages/server && go test ./...
