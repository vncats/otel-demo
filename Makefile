up:
	@docker compose -p movie -f docker-compose.yaml up -d --wait

down:
	@docker compose -p movie -f docker-compose.yaml down -v --remove-orphans

server:
	@OTEL_SDK_ENABLED=true \
	 OTEL_SDK_DEFAULT_CONFIG_FILE=./otel-sdk-config.yaml \
	 OTEL_RESOURCE_ATTRIBUTES=service.name=movie_service,service.version=1.1.2,deployment.environment=staging \
	 go run cmd/server/main.go

client:
	@go run cmd/client/main.go

.PHONY: up down server client