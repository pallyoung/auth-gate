.PHONY: all build run dev clean test

all: build

build:
	$(MAKE) -C packages/server build

run:
	$(MAKE) -C packages/server run

dev:
	$(MAKE) -C packages/server dev

clean:
	$(MAKE) -C packages/server clean

test:
	$(MAKE) -C packages/server test
