# Spec: Settings

## Objective
Ship `/settings` with separated user, system, and admin settings. User preferences change client UI and persist through an abstraction with local storage fallback. System status is read-only and safe for all users. Admin settings persist in PostgreSQL, record immutable audit history, validate cross-field constraints, and advertise only runtime behavior actually supported by running services.

## Tech Stack
- Next.js 15 / React 19 frontend
- Go HTTP API with chi
- PostgreSQL via pgx migrations
- Existing terminal workspace store for appearance and layout application

## Commands
- Backend test: `go test ./...`
- Frontend build: `npm run build`
- Frontend dev: `npm run dev`
- Migration: `go run ./cmd/migrate up`

## Project Structure
- `backend/internal/settings/`: runtime-safe settings schema, validation, reload metadata
- `backend/internal/httpapi/`: protected settings handlers
- `backend/internal/storage/`: PostgreSQL settings repository
- `backend/migrations/versioned/`: versioned schema migration
- `web/app/settings/`: responsive settings page
- `web/lib/settings.ts`: preference repository and API contract
- `web/components/settings/`: client settings UI

## Code Style
```go
if input.SignalConfirmScore < input.SignalSetupScore {
	return ValidationError("signal_confirm_score must be greater than or equal to signal_setup_score")
}
```

Use explicit allowlists for persisted setting keys. Never serialize environment values or credentials. Validate request bodies at handlers and again in settings service.

## Testing Strategy
- Go unit tests: validation, redaction, authorization, audit construction, runtime reload classification.
- Frontend build/type validation.
- Browser verification at mobile and desktop sizes when DevTools available.

## Boundaries
- Always: validate API input, role-check admin mutation, redact secrets, retain audit records, preserve unrelated worktree changes.
- Ask first: change existing auth provider, add dependency, edit `.env`, delete audit records.
- Never: put secrets in settings tables, API payloads, local storage, frontend environment, or normal UI.

## Success Criteria
- User settings available, applied, and show `Saved locally` when auth unavailable.
- System response contains status only, never secret configuration.
- Admin writes require admin role, validate settings, create version and audit records.
- Runtime metadata accurately states immediate/new-signal/resubscription/restart effect.
- Responsive settings UI works across supported viewports.

## Open Decisions
- Auth absent. Development adapter uses trusted reverse-proxy identity headers only when `APP_ENV=development`; production denies admin API until real authentication middleware injects identity.
