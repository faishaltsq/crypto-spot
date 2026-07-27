# Deployment notes

## Local deployment

Copy `.env.example` to `.env`, then start the Compose stack. The one-shot `migrate` service waits for PostgreSQL health, runs versioned migrations, then permits backend startup. Existing PostgreSQL volumes are retained and migrated in place.

## Migrations

`golang-migrate` owns schema history in PostgreSQL's `schema_migrations` table and uses PostgreSQL's migration lock. Migration SQL is immutable under `backend/migrations/versioned/`; its SHA-256 manifest is checked before database writes.

```bash
make migrate-up
make migrate-status
make migrate-version
make migrate-down
make migrate-repair
```

`down` rolls back only latest migration. A dirty migration state causes a non-zero exit. Inspect and repair affected schema, then run `make migrate-repair` to clear its dirty marker before retrying. Migration logs never print `DATABASE_URL`.

## Production changes

Before public deployment:

1. Put the web and backend behind HTTPS.
2. Restrict CORS and WebSocket origins.
3. Do not expose PostgreSQL or Redis ports publicly.
4. Use managed secrets for AI keys.
5. Pin container image digests.
6. Add database backups and TimescaleDB retention policies.
7. Add authentication before exposing user-specific notification settings.
8. Add Prometheus metrics and alerting for stale market data, reconnect loops, and order book resynchronization.

## Scaling

Start with a small liquid pair set. One process can handle the default five pairs. When expanding, partition pair ownership between ingestor instances. Use NATS JetStream, Redis Streams, or Kafka only after a measured bottleneck appears.

## Financial boundary

This repository contains no exchange-authenticated trading client and no create-order endpoint. Keep execution in a separate service if it is ever developed. Do not grant withdrawal permission to any API key.
