SHELL := /usr/bin/env bash -o pipefail
# This controls the location of the cache.
PROJECT := grpcsrv

JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION	= $(shell git rev-parse --short HEAD)
GIT_TAG		?= $(shell git describe --tags --abbrev=0)
SHORT_SHA ?= $(shell git rev-parse --short HEAD)
BUILD_DIR = "build"
GOLANGCI_LINT = $(BUILD_DIR)/golangci-lint

LDFLAGS		+= -s -w
LDFLAGS		+= -X github.com/apollo-hq/fleet-node-daemon/pkg/version.Version=$(GIT_TAG)
LDFLAGS		+= -X github.com/apollo-hq/fleet-node-daemon/pkg/version.Revision=$(GIT_REVISION)
LDFLAGS		+= -X github.com/apollo-hq/fleet-node-daemon/pkg/version.BuildDate=$(JOBDATE)

BUF_VERSION := 1.19.0
# If true, Buf is installed from source instead of from releases
BUF_INSTALL_FROM_SOURCE := false

### Everything below this line is meant to be static, i.e. only adjust the above variables. ###

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
# Buf will be cached to ~/.cache/buf-example.
CACHE_BASE := $(HOME)/.cache/$(PROJECT)
# This allows switching between i.e a Docker container and your local setup without overwriting.
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)
# The location where buf will be installed.
CACHE_BIN := $(CACHE)/bin
# Marker files are put into this directory to denote the current version of binaries that are installed.
CACHE_VERSIONS := $(CACHE)/versions

# Update the $PATH so we can use buf directly
export PATH := $(abspath $(CACHE_BIN)):$(PATH)
# Update GOBIN to point to CACHE_BIN for source installations
export GOBIN := $(abspath $(CACHE_BIN))
# This is needed to allow versions to be added to Golang modules with go get
export GO111MODULE := on

# BUF points to the marker file for the installed version.
#
# If BUF_VERSION is changed, the binary will be re-downloaded.
BUF := $(CACHE_VERSIONS)/buf/$(BUF_VERSION)
$(BUF):
	@rm -f $(CACHE_BIN)/buf
	@mkdir -p $(CACHE_BIN)
ifeq ($(BUF_INSTALL_FROM_SOURCE),true)
	$(eval BUF_TMP := $(shell mktemp -d))
	cd $(BUF_TMP); go get github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@rm -rf $(BUF_TMP)
else
	curl -sSL \
		"https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$(UNAME_OS)-$(UNAME_ARCH)" \
		-o "$(CACHE_BIN)/buf"
	chmod +x "$(CACHE_BIN)/buf"
endif
	@rm -rf $(dir $(BUF))
	@mkdir -p $(dir $(BUF))
	@touch $(BUF)

.DEFAULT_GOAL := local

.PHONY: build
build:
	cd cmd/fleet-node-daemon && CGO_ENABLED=0 GOOS=linux go build -mod vendor -ldflags "$(LDFLAGS)" -o fleet-node-daemon .

.PHONY: build-osx
build-osx:
	cd cmd/fleet-node-daemon && CGO_ENABLED=0 go build -mod vendor -ldflags "$(LDFLAGS)" -o fleet-node-daemon .

stresser:
	cd cmd/stresser && CGO_ENABLED=0 GOOS=linux go build -mod vendor -ldflags "$(LDFLAGS)" -o stresser .
	gcloud compute scp cmd/stresser/stresser instance-20240508-114427:/data/deliust/stresser --project deliust-staging

.PHONY: test
test:
	go test -mod vendor `go list ./... | egrep -v /tests/`

run: build
	./cmd/fleet-node-daemon/fleet-node-daemon

run-osx: build-osx
	./cmd/fleet-node-daemon/fleet-node-daemon

# Start virtual card reader that the daemon can connect to. This server has several HTTP APIs that can
# be used for integration testing and development

run-virtual-reader:
	cd cmd/virtual-card-reader && go install
	virtual-card-reader

cli:
	cd cmd/fncli && go install

fetch-certs:
	curl -L --remote-name --time-cond cacert.pem https://curl.se/ca/cacert.pem
	cp cacert.pem ca-certificates.crt

.PHONY: build-armhf
build-armhf:
	cd cmd/fleet-node-daemon && env CGO_ENABLED=0 GOARCH=arm GOOS=linux go build -mod vendor -ldflags="$(LDFLAGS)" -o release/fleet-node-daemon-linux-arm

# Currently not used in the normal build process
compress-arm:
	upx --brute cmd/fleet-node-daemon/release/fleet-node-daemon-linux-arm

# Build and push arm dev image
armhf: build-armhf
	docker build -t ghcr.io/apollo-hq/fleet-node-daemon:$(SHORT_SHA) -f Dockerfile.armhf .
	docker push ghcr.io/apollo-hq/fleet-node-daemon:$(SHORT_SHA)

arm: armhf

amd64:
	docker build -t quay.io/apollo-technologies/fleet-node-daemon-amd64:dev -f Dockerfile .

##############################
#         Bundler            #
##############################

.PHONY: build-bundler
build-bundler:
	cd cmd/fn-bundler && CGO_ENABLED=0 GOOS=linux go build -mod vendor -ldflags "$(LDFLAGS)" -o fn-bundler .

.PHONY: bundler
bundler:
	cd cmd/fn-bundler && go install

##############################
#           DEV              #
##############################

# deps allows us to install deps without running any checks.

.PHONY: deps
deps: $(BUF)

# local is what we run when testing locally.
# This does breaking change detection against our local git repository.

.PHONY: local
local-proto-lint: $(BUF)
	buf lint
	buf breaking --against '.git#branch=master'

proto:
	buf generate --template buf.gen.yaml

generate:
	go generate ./...

lint:
	$(GOLANGCI_LINT) run

arm-virtual-info-server:
	cd cmd/virtual-info-server && env CGO_ENABLED=0 GOARCH=arm GOOS=linux go build -ldflags="$(LDFLAGS)" -o release/virtual-info-server-arm
	docker build -t ghcr.io/apollo-hq/virtual-info-server:latest-arm -f Dockerfile.virtual-info-server.armhf .
	docker push ghcr.io/apollo-hq/virtual-info-server:latest-arm

arm-virtual-card-reader:
	cd cmd/virtual-card-reader && env CGO_ENABLED=0 GOARCH=arm GOOS=linux go build -ldflags="$(LDFLAGS)" -o release/virtual-card-reader-arm
	docker build -t ghcr.io/apollo-hq/virtual-card-reader:latest-arm -f Dockerfile.virtual-reader.armhf .
	docker push ghcr.io/apollo-hq/virtual-card-reader:latest-arm

##############################
#     Third-party tools      #
##############################

golangci-lint: ## Install golangci-lint
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BUILD_DIR) v1.27.0; \
	fi
