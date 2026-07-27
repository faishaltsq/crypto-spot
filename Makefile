.PHONY: init up down logs rebuild test smoke migrate migrate-up migrate-down migrate-status migrate-version migrate-repair

init:
	@test -f .env || cp .env.example .env

up: init
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=200

rebuild:
	docker compose build --no-cache

migrate:
	docker compose run --rm migrate $(ACTION)

migrate-up:
	docker compose run --rm migrate up

migrate-down:
	docker compose run --rm migrate down

migrate-status migrate-version:
	docker compose run --rm migrate status

migrate-repair:
	docker compose run --rm migrate repair

smoke:
	python scripts/smoke_check.py

test: smoke
	cd backend && go test ./internal/market ./internal/features ./internal/migration ./cmd/migrate
	cd ai-service && python -m unittest discover -s tests
