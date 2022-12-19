include includes.mk

PHONY: install run run-sub run-pub set-env unset-env

APPS ?= pub sub

.DEFAULT_GOAL := help

ifneq (,$(wildcard ./.env))
    include .env
    export
endif

install: ## download dependencies
	@go mod download > /dev/null >&1

build:	install ## build binary
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) build ;)

env-up:
	@docker-compose up -d
env-down:
	@docker-compose down -v

run: ## run
	@$(foreach APP, $(APPS), $(MAKE) -C $(APPS_DIR)/$(APP) run ;)

run-sub: ## run subscriber
	@$(MAKE) -C $(APPS_DIR)/sub run

run-pub: install ## run publisher
	@$(MAKE) -C $(APPS_DIR)/pub run


help:
	@grep -hE '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'