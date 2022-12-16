include includes.mk

PHONY: install run run-sub run-pub

APPS ?= pub sub

.DEFAULT_GOAL := help

install: ## download dependencies
	@go mod download > /dev/null >&1

env-up:
	@docker-compose up -d
env-down:
	@docker-compose down -v

run: ## run
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) run ;)

run-sub: ## run subscriber
	@$(MAKE) -C $(APPS_DIR)/sub run

run-pub: ## run publisher
	@$(MAKE) -C $(APPS_DIR)/pub run


help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'