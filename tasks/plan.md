# Implementation Plan: Paper Execution Simulation Integration

## Objective
Persist per-notional paper fills for confirmed signals, evaluate exit fills from live order book depth, then expose net outcome returns to performance APIs. No order execution, private exchange API, or trading UI is in scope.

## Design
- Version 7 adds `paper_execution_simulations`, unique on `(signal_id, notional)`.
- Entry uses ask-side order-book walk. Exit uses bid-side walk. Missing book is `INCOMPLETE`; depth shortfall is `PARTIAL_FILL`.
- Monetary values are USDT. Return decimals and percentages are separate API fields.
- Outcome horizon values carry gross/net returns. Performance aggregates persisted net 1h horizon values, excluding unavailable simulation.

## Tasks
- [ ] Add migration and simulation math tests.
- [ ] Persist entry simulations immediately after confirmed signal save.
- [ ] Update exits through outcome loop; preserve per-notional entry state.
- [ ] Add simulations to signal API/history/export and net performance summary.
- [ ] Verify migration, integration, build, review, commit.

## Risks
| Risk | Mitigation |
| --- | --- |
| Missing levels | Persist explicit incomplete status; do not create zero costs. |
| Insufficient levels | Persist fill/unfilled amount and partial status. |
| Duplicate scanner run | DB unique constraint and upsert protect entry records. |
| Stale outcome | Horizon JSON is updated from persisted simulation result, not raw price-only return. |

# Implementation Plan: AI Reviewer Safety Layer

## Overview
Replace existing loosely bounded reviewer integration with explicit review-only contracts. Keep rule engine as sole authority for candidate status and confirmation. Persist AI review audit data separately.

## Architecture Decisions
- Backend constructs strict feature-summary DTO. AI service rejects all fields outside schema.
- `AI_ENABLED=false` bypasses remote service and uses deterministic review. Missing remote-provider key becomes one `AI_PROVIDER_MISCONFIGURED` fallback, with no network retry.
- Provider calls remain isolated in AI service. Cache and circuit breaker live there because they control remote-provider cost/failure; backend also validates returned data before use.
- AI review may only provide metadata for non-blocked candidate evaluation. It cannot mutate feature status, thresholds, prices, blocked reasons, or signal execution.
- Persist reviews in a dedicated table; no raw request/provider response or secret fields stored.

## Task List

### Phase 1: Contracts And Defaults
- [ ] Task 1: Define safe environment defaults and typed reviewer input/output contracts.
  - Acceptance: four supported providers; no paid default; strict decision schema and feature allowlist.
  - Verify: focused Go/Python schema/config tests.
  - Files: config, schemas, provider helpers, `.env.example`.

- [ ] Task 2: Implement deterministic and remote provider behavior with fallback, redaction, retry, cache, and circuit breaker.
  - Acceptance: all failure modes produce deterministic fallback with generic reason codes; no secrets logged.
  - Verify: AI service unit tests for provider/failure/cache/breaker cases.
  - Files: AI service providers, service, tests.

### Checkpoint: Provider Boundary
- [ ] AI service tests pass.
- [ ] Default configuration contains no functional-looking provider key and cannot call paid provider.

### Phase 2: Backend Safety And Persistence
- [ ] Task 3: Restrict backend AI input, validate backend response, and preserve blocked signals/rules.
  - Acceptance: raw feature object never crosses boundary; blocked feature cannot be promoted; AI errors cannot stop scanner.
  - Verify: focused Go AI/signal tests.
  - Files: backend AI client, signal engine, config, tests.

- [ ] Task 4: Persist AI review audit fields through versioned migration and repository.
  - Acceptance: required review metadata stored without secrets; prompt/schema versions recorded.
  - Verify: migration checksum tests and storage tests/build.
  - Files: migration, checksum manifest, domain, storage, scanner flow.

### Checkpoint: End-to-End Review
- [ ] Full Go and Python tests pass.
- [ ] Backend build succeeds.
- [ ] No review path can place orders or access private Gate APIs.

## Risks And Mitigations
| Risk | Impact | Mitigation |
| --- | --- | --- |
| Existing signal engine couples AI decision to confirmation | High | Preserve eligibility checks before and after AI review; test blocked state cannot promote. |
| Provider output malformed or adversarial | High | Strict Pydantic/Go validation and deterministic fallback. |
| Provider outage/cost loop | High | One retry, TTL cache, circuit breaker, missing-key no-retry path. |
| Secret disclosure | High | Environment-only keys, allowlisted request DTO, redacted errors, no raw payload persistence. |
| Migration checksum enforcement | Medium | Update manifest only after migration SQL final and run migration tests. |

## Open Questions
- None. Store reviews separately and keep existing public signal shape unchanged unless current architecture requires relation exposure.
