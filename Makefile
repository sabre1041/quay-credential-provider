ROOT ?= $(shell pwd)
BINDIR      := $(ROOT)/bin
BINNAME     ?= quay-credential-provider
SHELL := /bin/bash
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOPROXY ?= $(shell go env GOPROXY)
LDFLAGS ?= -w -s

.PHONY: build
build:
	 GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOPROXY=$(GOPROXY) go build \
		-trimpath \
		-ldflags="$(LDFLAGS)" \
		-o=$(BINDIR)/$(BINNAME) \
		main.go

.PHONY: clean
clean:
	@rm -rf '$(BINDIR)'