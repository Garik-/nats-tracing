include ../../includes.mk

APP := pub
ID := 1

.PHONY: run build

build:
	@echo "=> building $(APP) binary"
	@$(GO_FLAGS) $(GO_LDFLAGS) $(GO) build -o $(BIN_DIR)/$(APP) .

run: build
	@echo "=> run $(APP) $(VERSION)"
	@$(BIN_DIR)/$(APP) --version=$(VERSION) --service=$(APP) --id=$(ID)