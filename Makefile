.PHONY: openapi-validate openapi-diff go-test sqlc-generate db-test-up db-test-down test-integration
.PHONY: lint-go format-go build-go check-go check-frontend check-all

OPENAPI_FILE := docs/api/teamflow-openapi.yaml

openapi-validate:
	@echo "Validating OpenAPI specification..."
	@pnpm exec redocly bundle $(OPENAPI_FILE) --output /tmp/teamflow-openapi-bundled.yaml > /dev/null 2>&1 && rm -f /tmp/teamflow-openapi-bundled.yaml && echo "✓ OpenAPI specification is valid"

openapi-diff:
	@echo "Checking OpenAPI breaking changes against origin/master..."
	@git fetch origin master:refs/remotes/origin/master 2>/dev/null || git fetch origin main:refs/remotes/origin/main 2>/dev/null || true
	@if [ ! -f "$(OPENAPI_FILE)" ]; then \
		echo "⚠ OpenAPI file not found, skipping diff check"; \
		exit 0; \
	fi
	@if command -v oasdiff >/dev/null 2>&1; then \
		: ; \
	else \
		echo "✗ oasdiff not found. Install it with:"; \
		echo "  go install github.com/oasdiff/oasdiff@latest"; \
		echo "  export PATH=\"$$PATH:$$(go env GOPATH)/bin\""; \
		exit 127; \
	fi
	@if git show origin/master:$(OPENAPI_FILE) > /tmp/openapi-base.yaml 2>/dev/null; then \
		BASE_BRANCH="origin/master"; \
	elif git show origin/main:$(OPENAPI_FILE) > /tmp/openapi-base.yaml 2>/dev/null; then \
		BASE_BRANCH="origin/main"; \
	else \
		echo "⚠ Warning: Could not find base version on origin/master or origin/main, skipping diff check"; \
		exit 0; \
	fi; \
	echo "Base: $$BASE_BRANCH"; \
	echo "Running oasdiff (go binary) ..."; \
	oasdiff breaking /tmp/openapi-base.yaml $(OPENAPI_FILE); \
	RC=$$?; \
	rm -f /tmp/openapi-base.yaml; \
	if [ $$RC -ne 0 ]; then \
		echo "✗ Breaking changes detected"; \
		exit $$RC; \
	fi; \
	echo "✓ No breaking changes detected"

sqlc-generate:
	cd apps/tasks && sqlc generate

go-test: sqlc-generate
	cd apps/projects && go test ./...
	cd apps/tasks && go test ./...

db-test-up:
	docker compose -f docker-compose.test.yml up -d --wait

db-test-down:
	docker compose -f docker-compose.test.yml down -v

test-integration:
	@set -e; \
	ROOT_DIR="$$(pwd)"; \
	trap '$(MAKE) -C "$$ROOT_DIR" db-test-down' EXIT; \
	$(MAKE) db-test-up; \
	cd apps/tasks && \
	DB_TEST_DSN="postgres://teamflow:teamflow@localhost:15432/teamflow_tasks_test?sslmode=disable" \
	go test -tags=integration ./... -count=1 -p 1

# Go lint and format
lint-go:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "✗ golangci-lint not found. Install it with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "  export PATH=\"\$$PATH:\$$(go env GOPATH)/bin\""; \
		exit 127; \
	fi
	@cd apps/projects && golangci-lint run ./...
	@cd apps/tasks && golangci-lint run ./...
	@echo "✓ Go lint passed"

format-go:
	@echo "Formatting Go code..."
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "✗ goimports not found. Install it with:"; \
		echo "  go install golang.org/x/tools/cmd/goimports@latest"; \
		echo "  export PATH=\"\$$PATH:\$$(go env GOPATH)/bin\""; \
		exit 127; \
	fi
	@cd apps/projects && goimports -w -local github.com/kumityou/teamflow .
	@cd apps/projects && go fmt ./...
	@cd apps/tasks && goimports -w -local github.com/kumityou/teamflow .
	@cd apps/tasks && go fmt ./...
	@echo "✓ Go code formatted"

build-go:
	@echo "Building Go services..."
	@cd apps/projects && go build -v ./cmd/...
	@cd apps/tasks && go build -v ./cmd/...
	@echo "✓ Go build succeeded"

# Integrated checks
check-go: lint-go build-go go-test
	@echo "✓ All Go checks passed"

check-frontend:
	@echo "Running Frontend checks..."
	@pnpm format:check
	@pnpm lint
	@pnpm build
	@echo "✓ All Frontend checks passed"

check-all: openapi-validate check-go check-frontend
	@echo "✓ All checks passed"
