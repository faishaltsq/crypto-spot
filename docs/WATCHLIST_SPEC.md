# Spec: Watchlist

## Objective
Provide user-owned watchlists at `/watchlist`. Preferences affect only local watchlist presentation and browser notifications. They never change market universe, ranks, tiers, signal thresholds, scanner selection, or subscriptions.

## Storage
Authentication is not available. Browser local storage is active storage and UI must say `Saved locally`. A typed repository keeps a future authenticated API implementation replaceable. PostgreSQL migration is preparation only; no unauthenticated API is exposed.

## Realtime
Watchlist reads existing global scanner and signal Zustand state. `GlobalRealtime` remains sole WebSocket owner. Changed scanner symbols update only their matching rendered rows through store state.

## Notification Policy
An enabled, unmuted pair can notify only when signal score, preferred timeframe, selected risk levels, selected signal types, and local-timezone quiet-hours all match. Browser permission remains required.

## Commands
Build: `cd web; npm run build`
E2E: `cd web; npx playwright test tests/watchlist.spec.ts`
Migration tests: `cd backend; go test ./internal/migration ./cmd/migrate`

## Boundaries
- Always: retain one global WebSocket, label local storage honestly, prevent duplicate symbols per watchlist.
- Ask first: authenticated ownership/API implementation, new dependency, notification delivery while browser is closed.
- Never: modify scanner universe, global rank/tier, score threshold, subscription tier, Compare, Settings, or AI reviewer files.

## Success Criteria
- Watchlists and pair preferences persist locally.
- CRUD, pair actions, filters, sorting, pinning, reorder, notes, tags, alerts, and quiet hours work.
- UI has loading, empty, unavailable, stale, storage, backend, and notification-permission states.
- Existing global socket is reused.
