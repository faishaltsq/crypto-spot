# Simulation Integration Tasks

- [ ] Per-notional execution migration and order-book math.
  - Verify: `go test ./internal/execution_simulation`.
- [ ] Runtime entry and outcome exit integration.
  - Verify: simulator is invoked after confirmed signal persistence.
- [ ] Persisted net performance/API integration.
  - Verify: outcome and performance tests; migration v7.

# AI Reviewer Safety Tasks

- [ ] Task 1: Safe AI contracts and defaults
  - Acceptance: supported provider enum, strict allowlisted DTOs, safe environment defaults.
  - Verify: focused contract tests.
  - Files: `ai-service/app/config.py`, `ai-service/app/schemas.py`, `backend/internal/config/config.go`, `.env.example`.

- [ ] Task 2: Provider resilience layer
  - Acceptance: deterministic fallback, key misconfiguration behavior, retry/cache/circuit breaker/redaction.
  - Verify: `cd ai-service; python -m unittest discover -s tests -v`.
  - Files: `ai-service/app/providers/`, `ai-service/app/service.py`, `ai-service/tests/`.

- [ ] Task 3: Backend review-only integration
  - Acceptance: feature summary only, strict response validation, no blocked override.
  - Verify: `cd backend; go test ./internal/ai ./internal/signals`.
  - Files: `backend/internal/ai/`, `backend/internal/signals/`, `backend/internal/config/`.

- [ ] Task 4: AI review audit persistence
  - Acceptance: required metadata stored without secrets or raw bodies.
  - Verify: `cd backend; go test ./...; go build ./cmd/server`.
  - Files: `backend/migrations/versioned/`, `backend/internal/domain/`, `backend/internal/storage/`, `backend/cmd/server/`.

- [ ] Task 5: Full verification and phase commit
  - Acceptance: full Go/Python tests pass, staged diff secret-scan clean, commit created.
  - Verify: repository commands above.
  - Files: only intended implementation/docs files.
