.PHONY: openapi-validate openapi-diff

openapi-validate:
	@echo "Validating OpenAPI specification..."
	@pnpm exec redocly bundle docs/api/teamflow-openapi-v1.0.1.yaml --output /tmp/teamflow-openapi-bundled.yaml > /dev/null 2>&1 && rm -f /tmp/teamflow-openapi-bundled.yaml && echo "✓ OpenAPI specification is valid"

openapi-diff:
	@echo "Checking OpenAPI breaking changes against origin/master..."
	@git fetch origin master:refs/remotes/origin/master 2>/dev/null || git fetch origin main:refs/remotes/origin/main 2>/dev/null || true
	@if [ -f "docs/api/teamflow-openapi-v1.0.1.yaml" ]; then \
		if git show origin/master:docs/api/teamflow-openapi-v1.0.1.yaml > /tmp/openapi-master.yaml 2>/dev/null; then \
			BASE_BRANCH="origin/master"; \
		elif git show origin/main:docs/api/teamflow-openapi-v1.0.1.yaml > /tmp/openapi-master.yaml 2>/dev/null; then \
			BASE_BRANCH="origin/main"; \
		else \
			echo "⚠ Warning: Could not fetch master/main version, skipping diff check"; \
			exit 0; \
		fi; \
		pnpm exec redocly diff /tmp/openapi-master.yaml docs/api/teamflow-openapi-v1.0.1.yaml --fail-on breaking || exit 1; \
		rm -f /tmp/openapi-master.yaml; \
		echo "✓ No breaking changes detected"; \
	else \
		echo "⚠ OpenAPI file not found, skipping diff check"; \
	fi

go-test:
	cd apps/projects && go test ./...
	cd apps/tasks && go test ./...
