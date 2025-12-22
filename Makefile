.PHONY: openapi-validate openapi-diff go-test

OPENAPI_FILE := docs/api/teamflow-openapi-v1.0.1.yaml

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

go-test:
	cd apps/projects && go test ./...
	cd apps/tasks && go test ./...
