unexport GOBIN

GO           ?= go
GOFMT        ?= $(GO)fmt
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU        := $(FIRST_GOPATH)/bin/promu
STATICCHECK  := $(FIRST_GOPATH)/bin/staticcheck
GOVENDOR     := $(FIRST_GOPATH)/bin/govendor
pkgs          = $(shell $(GO) list ./... | grep -v /vendor/)

PREFIX                  ?= $(shell pwd)
BIN_DIR                 ?= $(shell pwd)

STATICCHECK_IGNORE =

ifeq ($(GOHOSTARCH),amd64)
        ifneq ($(OS_detected),SunOS)
                # Only supported on amd64
                test-flags := -race
        endif
endif

all: format staticcheck unused build test

style:
	@echo ">> checking code style"
	@! $(GOFMT) -d $(shell find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

test:
	@echo ">> running tests"
	$(GO) test -short $(test-flags) $(pkgs)

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

staticcheck: $(STATICCHECK)
	@echo ">> running staticcheck"
	@$(STATICCHECK) -ignore "$(STATICCHECK_IGNORE)" $(pkgs)

unused: $(GOVENDOR)
	@echo ">> running check for unused packages"
	@$(GOVENDOR) list +unused

build: promu
	@echo ">> building binaries"
	@$(PROMU) build --prefix $(PREFIX)

tarball: promu
	@echo ">> building release tarball"
	@$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

promu:
	@echo ">> fetching promu"
	@GOOS= GOARCH= $(GO) get -u github.com/prometheus/promu

$(FIRST_GOPATH)/bin/staticcheck:
	@GOOS= GOARCH= $(GO) get -u honnef.co/go/tools/cmd/staticcheck

$(FIRST_GOPATH)/bin/govendor:
	@GOOS= GOARCH= $(GO) get -u github.com/kardianos/govendor

.PHONY: all style format build test vet tarball promu staticcheck $(FIRST_GOPATH)/bin/staticcheck govendor $(FIRST_GOPATH)/bin/govendor
