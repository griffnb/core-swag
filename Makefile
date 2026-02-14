# Makefile
# ---------------------------

# Default shell for executing commands
SHELL := /bin/bash


 
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":"}; {target=$$2; sub(/^[[:space:]]*/, "", target); desc=$$3; for(i=4; i<=NF; i++) desc=desc":"$$i; sub(/.*## /, "", desc); printf "\033[36m%-30s\033[0m %s\n", target, desc}'


# Include standard makes
include ./scripts/makes/core.mk

.PHONY: build
build:
	go build -o core-swag ./cmd/core-swag

.PHONY: install
install:
	go install ./cmd/core-swag

.PHONY: test
test:
	echo "mode: count" > coverage.out
	for PKG in $(PACKAGES); do \
		go test -v -covermode=count -coverprofile=profile.out $$PKG > tmp.out; \
		cat tmp.out; \
		if grep -q "^--- FAIL" tmp.out; then \
			rm tmp.out; \
			exit 1; \
		elif grep -q "build failed" tmp.out; then \
			rm tmp.out; \
			exit; \
		fi; \
		if [ -f profile.out ]; then \
			cat profile.out | grep -v "mode:" >> coverage.out; \
			rm profile.out; \
		fi; \
	done


.PHONY: install-core
install-core: ## Install private Go modules
	@bash -c '\
		export GOPRIVATE=github.com/griffnb/core/*; \
		export GH_TOKEN=$$(gh auth token); \
		go get github.com/griffnb/core/lib@latest \
	'


#CKB targets

.PHONY: ckb-init
ckb-init: ## Initialize CKB node
	@npx @tastehub/ckb init --force

.PHONY: ckb-reindex
ckb-reindex: ## Reindex CKB data
	@npx @tastehub/ckb status
	@npx @tastehub/ckb index

.PHONY: ckb-start
ckb-start: ## Start CKB node
	@npx @tastehub/ckb daemon start


.PHONY: test-project-1
test-project-1: ## Run tests for project 1
	@echo "Running generation for project 1..."
	@go install ./cmd/core-swag && cd /Users/griffnb/projects/Crowdshield/atlas-go && core-swag init -g "main.go" -d "./cmd/server,./internal/controllers,./internal/models" --parseInternal -pd -o "./swag_docs"


.PHONY: test-project-2
test-project-2: ## Run tests for project 2
	@echo "Running generation for project 2..."
	@go install ./cmd/core-swag && cd /Users/griffnb/projects/botbuilders/go-the-schwartz && core-swag init -g "main.go" -d "./cmd/server,./internal/controllers,./internal/models,./applications" --parseInternal -pd -o "./swag_docs"
