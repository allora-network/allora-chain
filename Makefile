BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# process build tags
LEDGER_ENABLED ?= true
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
	ifeq ($(OS),Windows_NT)
	GCCEXE = $(shell where gcc.exe 2> NUL)
	ifeq ($(GCCEXE),)
		$(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
	else
		build_tags += ledger
	endif
	else
	UNAME_S = $(shell uname -s)
	ifeq ($(UNAME_S),OpenBSD)
		$(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
	else
		GCC = $(shell command -v gcc 2> /dev/null)
		ifeq ($(GCC),)
			$(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
		else
			build_tags += ledger
		endif
	endif
	endif
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=allora \
	-X github.com/cosmos/cosmos-sdk/version.AppName=allorad \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
BUILDDIR ?= $(CURDIR)/build

###########
# Install #
###########

all: install

install:
	# @echo "--> ensure dependencies have not been modified"
	# @go mod verify
	# @go mod tidy
	# @echo "--> installing allorad"
	@go install $(BUILD_FLAGS) -mod=readonly ./cmd/allorad

init:
	./scripts/init.sh

build:
	mkdir -p $(BUILDDIR)/
	GOWORK=off go build -mod=readonly  $(BUILD_FLAGS) -o $(BUILDDIR)/ github.com/allora-network/allora-chain/cmd/allorad

build-local-edits:
	mkdir -p $(BUILDDIR)/
	go build -mod=readonly  $(BUILD_FLAGS) -o $(BUILDDIR)/ github.com/allora-network/allora-chain/cmd/allorad

lint:
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3 run --timeout=10m
	@go run ./linter/check-defer-close .
	@go run ./linter/fuzz-transitions

build-maprange-linter:
	@echo "--> Buiding maprange linter"
	cd linter/maprange && go build -o bin/maprange.so -buildmode=plugin maprange.go

maprange: build-maprange-linter
	@echo "--> Running maprange linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3 run --timeout=10m --config linter/maprange/.golangci-maprange.yml

.PHONY: all install build lint build-maprange-linter maprange
