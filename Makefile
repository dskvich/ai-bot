GOLANGCI_LINT_VERSION := 1.64.8
GOLANGCI_LINT := $(HOME)/go/bin/golangci-lint

.PHONY: app
app:
	docker-compose up -d app

.PHONY: db
db:
	docker-compose up -d db

.PHONY: lint install-lint run-lint

lint: install-lint run-lint

install-lint:
	@if [ -f "$(GOLANGCI_LINT)" ]; then \
		INSTALLED_VERSION=$$($(GOLANGCI_LINT) version --format short | cut -d' ' -f1); \
		if [ "$$INSTALLED_VERSION" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "golangci-lint version mismatch ($$INSTALLED_VERSION found, $(GOLANGCI_LINT_VERSION) required). Updating..."; \
			rm -f "$(GOLANGCI_LINT)"; \
			curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(HOME)/go/bin v$(GOLANGCI_LINT_VERSION); \
			echo "golangci-lint updated successfully."; \
		else \
			echo "golangci-lint version $(GOLANGCI_LINT_VERSION) is already installed."; \
		fi; \
	else \
		echo "golangci-lint not found, installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(HOME)/go/bin v$(GOLANGCI_LINT_VERSION); \
		echo "golangci-lint installed successfully."; \
	fi

run-lint:
	$(GOLANGCI_LINT) run