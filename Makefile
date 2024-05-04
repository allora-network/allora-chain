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

# Update the ldflags with the app, client & server names
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=allora \
	-X github.com/cosmos/cosmos-sdk/version.AppName=allorad \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'
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

lint:
	go vet ./...
	staticcheck ./...