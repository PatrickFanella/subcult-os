# ADR 0004: Keep Alpha Event Discovery Isolated Ahead of First Discovery Cut

## Status

Accepted

## Context

CONTEXT.md defines the First Discovery Cut as direct-link Public Event Page access rather than a public discovery index. The alpha product already exposes Event Discovery in `/discover` and in mobile for rehearsal, so the system needs a clear boundary without hiding or disabling discovery.

## Decision

Keep Event Discovery active in alpha, but isolate it as a late-stage Module so the direct-link Public Event Page stays the v1 Interface.

## Consequences

- Discovery can be narrowed, expanded, or disabled later without changing the Public Event Page.
- Event Discovery has an explicit seam for future search, moderation, and ranking work.
- Architecture reviews should treat Event Discovery as alpha-ahead-of-v1 rather than deleting it because it sits before the First Discovery Cut.
