PROVIDER_NAME=irmc-redfish
VERSION?=1.0.0
BINARY=terraform-provider-${PROVIDER_NAME}
OS_ARCH=linux_amd64
INSTALL_ROOT?=~/.terraform.d/plugins
HOSTNAME=registry.terraform.io
NAMESPACE=fujitsu

default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: lint
lint:
	golangci-lint run --fix

.PHONY: doc
doc:
	go generate

.PHONY: build
build:
	go install
	go build -o $(CURDIR)/bin/$(OS_ARCH)/${BINARY}_v$(VERSION)

.PHONY: install
install: build
	mkdir -p $(INSTALL_ROOT)/${HOSTNAME}/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS_ARCH}
	mv $(CURDIR)/bin/${OS_ARCH}/${BINARY}_v$(VERSION) $(INSTALL_ROOT)/${HOSTNAME}/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS_ARCH}

.PHONY: testacc
testacc:
	TF_ACC=1 TF_LOG=INFO go test -v ./...
