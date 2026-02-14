


# Include standard makes
include ./scripts/makes/core.mk


all: test build

.PHONY: build
build: deps
	go build -o core-swag ./cmd/core-swag

.PHONY: install
install: deps
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

