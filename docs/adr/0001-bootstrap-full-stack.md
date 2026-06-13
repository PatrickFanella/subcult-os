# ADR 0001: Bootstrap subcult-os as a Full-Stack Project

## Status

Accepted

## Context

`subcult-os` needs a minimal, runnable starting point for a Go API, Vite React TypeScript frontend, Postgres, Docker Compose, and Makefile-driven verification.

## Decision

Keep this repository as a minimal full-stack project with shared project conventions, docs structure, Open Pilot readiness, verification entry points, and local Docker infrastructure.

## Consequences

- The project starts from consistent process and documentation.
- Go, React, Postgres, and Docker choices are explicit instead of generic placeholders.
- The repo remains easy to audit and update.
