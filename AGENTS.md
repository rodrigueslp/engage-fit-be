# EngageFit Backend — Agent Guide

## Required context

Before changing architecture, domain rules, database schema, tenancy,
authentication, privacy, external effects, or operational behavior, read:

1. `docs/system-design.md` — current implemented system;
2. `docs/sdd/ARCH-001-modular-foundation.md` — target modular architecture;
3. the applicable document under `docs/sdd/specs/`;
4. `.ai/handoff.md` for the latest operational context when the task involves
   production or external providers.

An accepted spec describes the target. `docs/system-design.md` describes what
is currently implemented. Update the system design only after implementation
and validation.

## Repository boundaries

- Keep the backend as a modular monolith.
- Follow the existing domain → application → ports → adapters direction.
- Do not import GORM models or concrete repositories into domain/application.
- A module must not query another module's tables as an undocumented shortcut.
- Keep SaaS `billing` separate from student `memberbilling`.
- Preserve `box_id` isolation while organization/unit migration is incomplete.
- Never perform identity merges based only on names.

## Working safely

- Preserve unrelated and pre-existing changes.
- Work in the branch/worktree assigned to the task.
- Do not push, deploy, change production data, or enable real sends unless the
  user explicitly authorizes it.
- Do not record credentials, tokens, full CPF, phone, email, message bodies, or
  raw provider payloads in logs, fixtures, docs, or responses.
- Migrations are immutable after publication and must be additive whenever
  possible.
- The integration owner assigns the final migration number before merge.
- External effects require feature flags, idempotency, auditability, timeouts,
  and safe errors.

## Implementation expectations

- Prefer vertical slices that include domain, persistence, use case, HTTP, and
  tests when those layers are in scope.
- Keep compatibility paths explicit and temporary; document their removal
  condition.
- Repositories and handlers must enforce tenant/unit scope even when IDs are
  already known.
- Changes to privacy-covered entities must update export, anonymization,
  suppression, and retention behavior as applicable.
- Do not mark a spec `validated` until its criteria are demonstrated.

## Verification

Run checks proportional to the change. The normal backend baseline is:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
git diff --check
```

Schema or critical-flow changes also require:

- all migrations on an empty PostgreSQL database;
- a second migrator run with zero applied migrations;
- PostgreSQL integration tests with `TEST_DATABASE_URL`;
- tenant isolation and compatibility regression tests;
- race detector in the supported Linux/container environment when relevant.

Report commands executed, skipped checks, remaining risks, and changed files.
