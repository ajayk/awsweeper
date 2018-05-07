TEST?=$$(go list ./... | grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
PKG_LIST := $(shell go list ./... | grep -v vendor)

default: build

.PHONY: build
build:
	go install

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: dep
dep:
	dep ensure

.PHONY: fmt
fmt: ## Run go fmt on all files except vendor
	gofmt -w -l $(GOFMT_FILES) .

.PHONY: test
test:
	go clean -testcache ${PKG_LIST}
	go test -short --race ${PKG_LIST}

.PHONY: test-all
test-all: generate ## Run all tests including --race
	go clean -testcache ${PKG_LIST}
	go test --race -v ${PKG_LIST}

.PHONY: testacc
testacc:
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m