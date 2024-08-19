#!/usr/bin/make -f
.DEFAULT_GOAL := proto-all

DOCKER := $(shell which docker)

#################
###   Build   ###
#################

build:
	@echo "--> Check the code compiles"
	@go build ./...

test:
	@echo "--> Running tests"
	go test -v ./...

.PHONY: build test

##################
###  Protobuf  ###
##################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating protobuf files..."
	@$(protoImage) sh ./scripts/protocgen.sh
	@go mod tidy

proto-format:
	@$(protoImage) buf format proto/ -w

proto-lint:
	@$(protoImage) buf lint proto/ --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking proto/ --against $(HTTPS_GIT)#branch=main

.PHONY: proto-all proto-gen proto-format proto-lint proto-check-breaking
