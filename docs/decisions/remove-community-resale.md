# ADR 002: Remove Community Posts & Resale features as core

- **Status**: Accepted
- **Decision Date**: 2026-06-28
- **Decision Maker**: Duck

## 1. Context

The community post feature, second-hand selling (resale), and handing over second-hand items between users (transfer item) generate many friction points in the actual experience and dilute the app's positioning as a smart personal digital wardrobe.

## 2. Decision

Completely remove all P2P transaction flows and social feed from the new MVP core. Store all old specification documents in the `docs/archive/` folder for reference when needed, and do not expand or maintain code related to this part in the near future.

## 3. Consequences

- Positive: Minimizes codebase complexity, focuses resources on developing the B2B segment.
- Negative: Some code sections related to community and resale will be cleaned up or frozen (deprecated).
