.PHONY: help dev test install
.DEFAULT_GOAL := help

help: ## Show available targets
	@printf '\033[1mUsage:\033[0m make \033[36m<target>\033[0m\n\n'
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{ printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }'

dev: ## Run with live reload (air)
	go run github.com/air-verse/air@latest -c .air.toml

test: ## Run tests
	go test ./...

install: ## Build and install colimui into GOBIN (~/go/bin)
	go install .
