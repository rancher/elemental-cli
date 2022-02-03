GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_TAG = $(shell git describe --tags 2>/dev/null || echo "v0.0.1" )

PKG        := ./...
LDFLAGS    := -w -s
LDFLAGS += -X "github.com/rancher-sandbox/elemental/internal/version.version=${GIT_TAG}"
LDFLAGS += -X "github.com/rancher-sandbox/elemental/internal/version.gitCommit=${GIT_COMMIT}"



build:
	go build -ldflags '$(LDFLAGS)' -o bin/

vet:
	go vet ${PKG}

fmt:
	go fmt ${PKG}

test:
	go install github.com/onsi/ginkgo/v2/ginkgo
	ginkgo --label-filter !root --fail-fast --slow-spec-threshold 30s --race --covermode=atomic --coverprofile=coverage.txt -p -r ${PKG}

test_root:
ifneq ($(shell id -u), 0)
	@echo "This tests require root/sudo to run."
	@exit 1
else
	go install github.com/onsi/ginkgo/v2/ginkgo
	ginkgo --label-filter root --fail-fast --slow-spec-threshold 30s --race --covermode=atomic --coverprofile=coverage_root.txt -p -r ${PKG}
endif

license-check:
	@.github/license_check.sh

lint: fmt vet

all: lint test build