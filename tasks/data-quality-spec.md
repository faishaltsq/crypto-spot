# Spec: Data Quality Route

## Objective
Create `/data-quality` App Router route using live backend quality reports. Show pair health without fabricated scores or converting absent values to zero.

## Commands
- `cd web && npm run build`
- `cd backend && go test ./internal/quality ./internal/httpapi`

## Structure
- `web/app/data-quality/` owns route, loading, error boundary.
- `web/lib/data-quality.ts` owns typed API parsing/filtering helpers.

## Testing
- Route build confirms direct route registration.
- Browser checks desktop/mobile render against backend responses.

## Boundaries
- Always: preserve null/unknown values, use one global WebSocket indirectly through existing app connection, keep filtering client-side over bounded API reports.
- Never: hardcode quality scores, alter signal engine, create per-row WebSockets, change unrelated features.
