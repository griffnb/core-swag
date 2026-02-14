.PHONY: tidy
tidy: ## Install private Go modules
	@bash -c '\
		export GOPRIVATE=github.com/griffnb/core/*; \
		export GH_TOKEN=$$(gh auth token); \
		go mod tidy \
	'

