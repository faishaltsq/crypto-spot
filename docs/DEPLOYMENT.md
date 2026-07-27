# Deployment notes

## Local deployment

Copy `.env.example` to `.env`, then start the Compose stack. The initial database script runs only when the PostgreSQL volume is created for the first time.

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
