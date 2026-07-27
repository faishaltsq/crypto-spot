# Simulation Integration Tasks

- [ ] Per-notional execution migration and order-book math.
  - Verify: `go test ./internal/execution_simulation`.
- [ ] Runtime entry and outcome exit integration.
  - Verify: simulator is invoked after confirmed signal persistence.
- [ ] Persisted net performance/API integration.
  - Verify: outcome and performance tests; migration v7.
