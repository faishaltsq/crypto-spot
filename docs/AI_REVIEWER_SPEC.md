# Spec: AI Reviewer Safety Layer

## Objective
Add a review-only AI layer for precomputed SPOT paper-signal candidates. Rule and signal engines remain authoritative. AI never generates prices, changes thresholds, creates orders, accesses Gate private APIs, accesses trading credentials, promotes `BLOCKED` to `CONFIRMED`, or removes blocked reasons.

## Tech Stack
- Go backend and PostgreSQL for scanner, signal lifecycle, persistence, and backend-side review request construction.
- Python FastAPI AI service for provider adapters, schema validation, cache, fallback, and circuit breaker.
- Existing Go and Python standard test tooling; no new provider SDK dependency.

## Commands
- Backend tests: `cd backend; go test ./...`
- AI tests: `cd ai-service; python -m unittest discover -s tests -v`
- Backend build: `cd backend; go build ./cmd/server`
- AI service smoke test: `cd ai-service; python -c "from app.main import app; print(app.title)"`

## Project Structure
- `backend/internal/ai/`: allowlisted feature summary construction, client validation, and deterministic fallback.
- `backend/internal/signals/`: signal eligibility remains rule-owned; AI only annotates eligible candidates.
- `backend/internal/storage/`: persists review audit fields without raw secrets or raw provider payloads.
- `backend/migrations/versioned/`: immutable migration for AI review records.
- `ai-service/app/`: provider adapters, strict request/output schemas, cache, circuit breaker, and redacted errors.
- `ai-service/tests/`: unit tests for configured modes, schema failures, retry/fallback, cache, breaker, safety, and persistence contract.

## Code Style
```go
// AI is advisory. Rule eligibility and blocked reasons are immutable here.
if feature.Status == "BLOCKED" {
    return deterministic(feature, "blocked")
}
```

Use typed boundary DTOs, explicit allowlists, immutable signal blocking state, bounded retries, and generic error codes. Never log request bodies, response bodies, or API keys.

## Testing Strategy
- Go unit tests cover summary allowlisting, response validation, blocked-signal non-promotion, fallback, and review persistence data.
- Python unit tests cover all four providers, missing keys, valid/invalid structured output, timeout/retry, cache hit/miss and duplicate requests, circuit transitions, secret redaction, and no-key startup.
- Full backend and AI test commands run before phase commit.

## Boundaries
- Always: read keys only from environment; validate provider output; use deterministic fallback; preserve rule thresholds and blocked reasons; record provider errors with redacted codes only.
- Ask first: adding external provider dependencies, changing public API response shapes, changing trading or Gate connectivity, or storing new sensitive data.
- Never: paid provider by default; placeholder key resembling a credential; raw order-book deltas/trade history/secrets in prompts; private Gate API; trading credential access; order placement; AI-driven promotion of a blocked signal.

## Success Criteria
- Defaults are `AI_ENABLED=false`, `AI_PROVIDER=deterministic`, empty `DEEPSEEK_API_KEY`/`GROK_API_KEY`, timeout `10`, retries `1`, cache TTL `300`, breaker failures `5`, reset `60`.
- Disabled, missing-key, timeout, invalid response, and provider errors keep scanner and signal engine running through deterministic fallback.
- Only feature-summary fields listed in request are sent to AI service/provider.
- Output accepts only `CONFIRM`, `REJECT`, `WAIT`, or `UNAVAILABLE`; confidence is labeled `AI review confidence` in exposed data.
- Review cache key includes pair, timeframe, feature version, feature snapshot, prompt version, provider, and model.
- Circuit breaker implements CLOSED, OPEN, and single-probe HALF_OPEN behavior.
- Review persistence includes required audit fields, no secret/raw prompt/raw provider body.

## Open Questions
- None. AI review remains advisory and stored as a separate review record rather than changing signal lifecycle.
