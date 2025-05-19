ROOT ?= $(shell pwd)
BINDIR      := $(ROOT)/bin
BINNAME     ?= quay-credential-provider
SHELL := /bin/bash
GOEXE ?= go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOPROXY ?= $(shell go env GOPROXY)
LDFLAGS ?= -w -s
ARCHITECTURES=amd64 arm64
PLATFORMS=darwin linux windows

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

.PHONY: cross
cross:
	$(foreach GOOS, $(PLATFORMS),\
		$(foreach GOARCH, $(ARCHITECTURES), $(shell export CGO_ENABLED=0; export GOOS=$(GOOS); export GOARCH=$(GOARCH); \
	$(GOEXE) build -trimpath -ldflags "$(LDFLAGS)" -o=$(BINDIR)/$(BINNAME)-$(GOOS)-$(GOARCH) main.go)))

