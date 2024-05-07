.PHONY: run

# Golang Flags
GOFLAGS ?= $(GOFLAGS:)
GO=go

run:
	$(GO) run $(GOFLAGS) $(GO_LINKER_FLAGS) ./cmd/main.go
