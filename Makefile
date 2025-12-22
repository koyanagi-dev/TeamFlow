.PHONY: openapi-validate

openapi-validate:
	@echo "Validating OpenAPI specification..."
	@pnpm exec redocly bundle docs/api/teamflow-openapi.yaml --output /tmp/teamflow-openapi-bundled.yaml > /dev/null 2>&1 && rm -f /tmp/teamflow-openapi-bundled.yaml && echo "✓ OpenAPI specification is valid"

go-test:
	cd apps/projects && go test ./...
	cd apps/tasks && go test ./...
