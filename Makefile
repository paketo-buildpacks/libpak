# Go parameters
GOCMD?=go
PACKAGE_BASE=github.com/buildpacks/libpak/v2

all: test

format:
	@echo "> Formating code..."
	$(GOCMD) tool goimports -l -w -local ${PACKAGE_BASE} .

lint:
	@echo "> Linting code..."
	$(GOCMD) tool golangci-lint run -c golangci.yaml

test: format lint
	$(GOCMD) test ./...
