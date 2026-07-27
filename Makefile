.PHONY: init up down logs rebuild test smoke

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

smoke:
	python scripts/smoke_check.py

test: smoke
	cd backend && go test ./internal/market ./internal/features
	cd ai-service && python -m unittest discover -s tests
