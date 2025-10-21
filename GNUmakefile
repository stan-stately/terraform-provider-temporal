default: build

# Build the provider
.PHONY: build
build:
	go build -o build/terraform-provider-temporal

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Run tests
.PHONY: test
test:
	go test ./...

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

