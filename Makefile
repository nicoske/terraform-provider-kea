GO ?= go
GOFMT_FILES ?= $$(find . -name '*.go' | grep -v vendor)
TF_LOG ?= error

KEA_DHCP4_ADDRESS ?= http://localhost:8000
KEA_DHCP4_HTTP_USERNAME ?=
KEA_DHCP4_HTTP_PASSWORD ?=
KEA_DHCP4_TLS_ADDRESS ?= https://localhost:8443

KEA_MATRIX_FILE := testing/kea-matrix.json
KEA_TEST_VERSIONS ?= $(shell jq -r '.[].version' $(KEA_MATRIX_FILE))

GO_VERSION := 1.26.0

default: build

build:
	$(GO) build -v

test:
	$(GO) get -t ./...
	$(GO) test -parallel $$(nproc) -race -timeout 60m -v ./...

testacc:
	KEA_DHCP4_ADDRESS=$(KEA_DHCP4_ADDRESS) \
	KEA_DHCP4_HTTP_USERNAME=$(KEA_DHCP4_HTTP_USERNAME) \
	KEA_DHCP4_HTTP_PASSWORD=$(KEA_DHCP4_HTTP_PASSWORD) \
	KEA_DHCP4_TLS_ADDRESS=$(KEA_DHCP4_TLS_ADDRESS) \
	TF_LOG=$(TF_LOG) TF_ACC=1 \
	$(GO) test -parallel 4 -v -race -count=1 $(TESTARGS) -timeout 60m ./internal/...

testacc-matrix:
	@for v in $(KEA_TEST_VERSIONS); do ./testing/testacc.sh $$v || exit 1; done

generate:
	cd tools && $(GO) generate

vet:
	@echo "$(GO) vet ."
	@$(GO) vet $$($(GO) list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt-go:
	gofmt -w $(GOFMT_FILES)

fmt-tf:
	@if command -v terraform > /dev/null; then \
		echo "==> Running terraform fmt"; \
		terraform fmt -recursive; \
	elif command -v tofu > /dev/null; then \
		echo "==> Running tofu fmt"; \
		tofu fmt -recursive; \
	else \
		echo "Missing \"terraform\" command, not formatting .tf" >&2; exit 1; \
	fi

fmt: fmt-go fmt-tf

fmtcheck-go:
	@echo "==> Checking that code complies with gofmt requirements..." ; \
	files=$$(find . -name '*.go') ; \
	gofmt_files=$$(gofmt -l $$files); \
	if [ -n "$$gofmt_files" ]; then \
		echo 'gofmt needs running on the following files:'; \
		echo "$$gofmt_files"; \
		echo "You can use the command: \`make fmt\` to reformat code."; \
		exit 1; \
	fi

fmtcheck-tf:
	@if command -v terraform > /dev/null; then \
		echo "==> Running terraform fmt check"; \
		terraform fmt -recursive -check -diff; \
	elif command -v tofu > /dev/null; then \
		echo "==> Running tofu fmt check"; \
		tofu fmt -recursive -check -diff; \
	else \
		echo "Missing \"terraform\" command, not checking .tf format" >&2; exit 1; \
	fi

fmtcheck: fmtcheck-go fmtcheck-tf

lint: vet
	@if command -v golangci-lint > /dev/null; then \
		echo "==> Running golangci-lint"; \
		golangci-lint run --timeout 5m; \
	else \
		echo "Missing \"golangci-lint\" command, not linting .go" >&2; exit 1; \
	fi

static-analysis: lint fmtcheck

update-gomod:
	$(GO) get -t -v -u ./...
	$(GO) mod tidy -go=$(GO_VERSION)
	$(GO) get toolchain@none
	cd tools
	$(GO) get -t -v -u ./...
	$(GO) mod tidy -go=$(GO_VERSION)
	$(GO) get toolchain@none
	@echo "Dependencies updated"

.PHONY: build test testacc testacc-matrix vet fmt fmt-go fmt-tf fmtcheck fmtcheck-go fmtcheck-tf generate lint static-analysis update-gomod
