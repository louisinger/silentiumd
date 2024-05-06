.PHONY: proto proto-lint

## proto: compile proto stubs
proto: proto-lint
	@echo "Compiling stubs..."
	@docker run --rm --volume "$(shell pwd):/workspace" --workdir /workspace buf generate

## proto-lint: lint protos
proto-lint:
	@echo "Linting protos..."
	@docker build -q -t buf -f buf.Dockerfile . &> /dev/null
	@docker run --rm --volume "$(shell pwd):/workspace" --workdir /workspace buf lint

build:
	@echo "Building..."
	@bash ./scripts/build.sh

build-all:
	@echo "Building all..."
	@bash ./scripts/build-all.sh