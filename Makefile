.PHONY: all build run dev clean

all: build

build: web-build
	go build -o bin/auth-gate ./cmd/server

run: build
	./bin/auth-gate

dev:
	DEBUG=true go run ./cmd/server

web-deps:
	cd web && npm install

web-build: web-deps
	cd web && npm run build

clean:
	rm -rf bin web/dist web/node_modules data
*** End Patch
