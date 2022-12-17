GO = go
BIN ?= bin
BIN_DIR ?= $(join $(dir $(lastword $(MAKEFILE_LIST))), $(BIN))
APPS_DIR ?= cmd
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_HASH ?= $(shell git rev-parse --short HEAD)
GIT_TAG_HASH ?=

VERSION = $(GIT_BRANCH)-$(GIT_HASH)