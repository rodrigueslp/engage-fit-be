# EngageFit Backend — Agent Guide

## Feature workflow

When the user requests a feature, improvement, or fix, carry it through
discovery, planning, implementation, and proportional testing. The direct user
request is sufficient authorization for local development. Do not require a
new SDD document, a `ready` status, or a separate implementation prompt.

Use reasonable assumptions for routine details and keep working. Ask the user
only when a missing product choice would materially change the result or when
an external, destructive, or production action needs new authorization.

Existing documents under `docs/sdd/` are preserved references, not workflow
gates. If an existing spec overlaps the requested feature, use the parts that
remain useful and prefer the user's current direction when they differ.

When the user asks to implement, continue, or finish a numbered EngageFit
spec, use `$engagefit-spec-executor`. For an umbrella initiative, preserve
validated slices and execute every remaining slice; for an explicitly named
slice, execute only that slice unless the user asks for the rest.

## Required context

Before changing architecture, domain rules, database schema, tenancy,
authentication, privacy, external effects, or operational behavior, inspect
the relevant parts of:

1. `docs/system-design.md` — current implemented system;
2. `docs/sdd/ARCH-001-modular-foundation.md` — target modular architecture;
3. any applicable document under `docs/sdd/specs/`, when one exists;
4. `.ai/handoff.md` for the latest operational context when the task involves
   production or external providers.

The user's current request describes the target. Existing specs provide prior
decisions and constraints; `docs/system-design.md` describes what is currently
implemented. Update the system design after a material implementation and its
validation.

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
- Plan as part of the task and implement once the relevant behavior and
  boundaries are understood; a separate planning artifact is optional.
- Keep compatibility paths explicit and temporary; document their removal
  condition.
- Repositories and handlers must enforce tenant/unit scope even when IDs are
  already known.
- Changes to privacy-covered entities must update export, anonymization,
  suppression, and retention behavior as applicable.
- If a spec is updated, do not mark it `validated` until its criteria are
  demonstrated. Routine feature work does not require creating or updating a
  spec.

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
