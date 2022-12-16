GO = go
BIN ?= bin
BIN_DIR ?= $(join $(dir $(lastword $(MAKEFILE_LIST))), $(BIN))
APPS_DIR ?= cmd
