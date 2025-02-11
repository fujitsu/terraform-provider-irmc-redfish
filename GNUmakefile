PROVIDER_NAME=irmc-redfish
VERSION?=0.0.1
BINARY=terraform-provider-${PROVIDER_NAME}
OS_ARCH=linux_amd64
INSTALL_ROOT?=~/.terraform.d/plugins
HOSTNAME=registry.terraform.io
NAMESPACE=fujitsu
OLD_PROVIDER_VERSION=0.0.0
NEW_PROVIDER_VERSION=${VERSION}

default: testacc

# Run acceptance tests
# to run choosen tests: TESTARGS="-run *TestName*" make testacc
.PHONY: testacc
testacc:
	TF_ACC=1 TF_LOG=INFO go test ./... $(TESTARGS) -timeout 120m -count=1

.PHONY: lint
lint:
	golangci-lint run --fix

.PHONY: doc
doc:
	go generate

.PHONY: change_examples_provider_version
change_examples_provider_version:
	find . -type f -name "provider.tf" -print0 | xargs -0 sed -i'' -e 's/${OLD_PROVIDER_VERSION}/${NEW_PROVIDER_VERSION}/g'

.PHONY: fmt
fmt:
	gofmt -w internal/

.PHONY: build
build:
	go install
	go build -o $(CURDIR)/bin/$(OS_ARCH)/${BINARY}_v$(VERSION)

.PHONY: install
install: build
	mkdir -p $(INSTALL_ROOT)/${HOSTNAME}/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS_ARCH}
	mv $(CURDIR)/bin/${OS_ARCH}/${BINARY}_v$(VERSION) $(INSTALL_ROOT)/${HOSTNAME}/${NAMESPACE}/${PROVIDER_NAME}/${VERSION}/${OS_ARCH}
