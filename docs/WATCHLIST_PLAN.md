# Watchlist Plan

## Architecture
- `web/lib/watchlist.ts`: types, local repository, validation, ordering, notification policy.
- `web/app/watchlist/page.tsx`: page-local UI state; consumes existing market store only.
- `web/components/GlobalRealtime.tsx`: evaluates local watchlist notification policy on existing `signal.new` event.
- `backend/migrations/versioned/009_*`: future authenticated persistence schema only.

## Tasks
- [ ] Add data contract, local persistence, notification policy, migration.
  - Verify: focused browser tests cover repository-facing UI actions and quiet hours.
- [ ] Build responsive watchlist selector, summary, filters, table, signal views, alert settings.
  - Verify: desktop/mobile Playwright screenshots and page interactions.
- [ ] Connect existing global realtime notification path without another socket.
  - Verify: one WebSocket assertion; notification policy test.
- [ ] Run build, browser tests, migration tests, review, commit.

## Risks
| Risk | Mitigation |
| --- | --- |
| Browser storage blocked | Surface `Local storage unavailable`; keep transient in-memory state. |
| Scanner unavailable | Preserve preferences and render `Pair unavailable`. |
| Notification policy leaks global changes | Read-only policy receives signal and stored preference; never writes market state. |
