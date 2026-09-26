# SPDX-License-Identifier: AGPL-3.0-only

# The checks CI runs are the ones to run before pushing (CLAUDE.md).

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

PYTHON ?= python3
T := bin/tools

# The workspace modules, and the package patterns that cover all of them.
MODULES := . spec core adapters/postgres adapters/gcp adapters/mail conformance
PKGS := ./... ./spec/... ./core/... ./adapters/postgres/... ./adapters/gcp/... ./adapters/mail/... ./conformance/...

# Each tool lives in its own module under tools/ (spec, section 4.3).
TOOLS := \
	golangci-lint=github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
	staticcheck=honnef.co/go/tools/cmd/staticcheck \
	gosec=github.com/securego/gosec/v2/cmd/gosec \
	govulncheck=golang.org/x/vuln/cmd/govulncheck \
	gitleaks=github.com/zricethezav/gitleaks/v8 \
	actionlint=github.com/rhysd/actionlint/cmd/actionlint \
	ko=github.com/google/ko

ZIZMOR := tools/.venv/bin/zizmor
WORKFLOWS := $(wildcard .github/workflows/*.yml)

.PHONY: help
help: ## List the targets
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

.PHONY: tools
tools: ## Build the pinned tools into bin/tools
	@for entry in $(TOOLS); do \
		name=$${entry%%=*}; pkg=$${entry#*=}; \
		GOWORK=off go build -C tools/$$name -o $(CURDIR)/$(T)/$$name $$pkg; \
	done
	@GOWORK=off go build -C tools/checks -o $(CURDIR)/$(T)/ ./cmd/...

$(ZIZMOR): tools/requirements.txt
	$(PYTHON) -m venv tools/.venv
	tools/.venv/bin/python -m pip install --quiet --disable-pip-version-check --require-hashes --no-deps -r tools/requirements.txt
	@touch $@

.PHONY: check
check: tools $(ZIZMOR) ## vet, staticcheck, golangci-lint, gosec, govulncheck, licences, exceptions, secrets, workflows
	go vet $(PKGS)
	cd tools/checks && GOWORK=off go vet ./...
	$(T)/staticcheck $(PKGS)
	@mkdir -p dist/sarif
	$(T)/golangci-lint run --config .golangci.yml --output.text.path=stdout --output.sarif.path=dist/sarif/golangci-lint.sarif $(PKGS)
	cd tools/checks && GOWORK=off $(CURDIR)/$(T)/golangci-lint run --config $(CURDIR)/.golangci.yml ./...
	@for m in $(MODULES); do echo "gosec $$m"; (cd $$m && $(CURDIR)/$(T)/gosec -quiet ./...); done
	@for m in $(MODULES); do echo "govulncheck $$m"; (cd $$m && $(CURDIR)/$(T)/govulncheck ./...); done
	$(T)/spdxcheck
	$(T)/trivyignorecheck .trivyignore.yaml
	$(T)/gitleaks git --redact --no-banner .
ifneq ($(WORKFLOWS),)
	$(T)/actionlint
	$(ZIZMOR) --offline .github/workflows
endif

.PHONY: test
test: ## Unit tests with the race detector, across the workspace and tools/checks
	go test -race -cover $(PKGS)
	cd tools/checks && GOWORK=off go test -race -cover ./...

.PHONY: integration
integration: ## Tests behind the integration tag (a real PostgreSQL from F4)
	go test -race -tags integration $(PKGS)

.PHONY: compile
compile: ## Compile every workspace package (CodeQL's build)
	go build $(PKGS)

BUILD_ID := $(shell git describe --tags --always --dirty --abbrev=12 2>/dev/null || echo unknown)
LDFLAGS := -X github.com/aleogr/identity/internal/buildinfo.version=$(BUILD_ID)

.PHONY: build
build: ## Build bin/identity with the build identifier
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/identity ./cmd/identity

.PHONY: run
run: build ## Build and run identity serve
	bin/identity serve

E2E_VENV := e2e/.venv

.PHONY: e2e-deps
e2e-deps: ## Create the end-to-end virtual environment (Playwright 1.56.0)
	$(PYTHON) -m venv $(E2E_VENV)
	$(E2E_VENV)/bin/python -m pip install --quiet --disable-pip-version-check --require-hashes --no-deps -r e2e/requirements.txt

.PHONY: e2e
e2e: build ## Run the end-to-end suite against bin/identity
	cd e2e && .venv/bin/python -m pytest

.PHONY: image
image: tools ## Build the container image into dist/identity-image.tar (no push)
	@mkdir -p dist
	BUILD_ID=$(BUILD_ID) KO_DOCKER_REPO=identity.local/identity $(T)/ko build --bare --push=false \
		--tarball=dist/identity-image.tar ./cmd/identity
