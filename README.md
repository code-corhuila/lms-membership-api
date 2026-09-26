# lms-membership-api

> Membership bounded context: members, plans, membership status

Part of the **LMS Library** distributed system — team `lms-library`, Grupo 2.
Governance and documentation live in [`library-docs`](https://github.com/code-corhuila/library-docs).

Go REST API implementing the Membership bounded context
(`library-docs/02-domain/domain-map.md`): owns student records — who is authorized to borrow,
and their suspension status. Hexagonal Architecture — see
`library-docs/05-architecture/decisions/records/ADR-002-hexagonal-modular-monolith.md`.

This service owns the `students` table exclusively — but the schema itself lives in
[`lms-membership-db`](https://github.com/code-corhuila/lms-membership-db), not here. It
validates the JWT issued by `lms-access-api` using the shared `JWT_SECRET` — no call to
access-service is needed to check a token.

Deactivating a student (HU-03) needs to know whether they have active loans — data that lives
in circulation-service's own database. `internal/infrastructure/circulation/client.go` calls
circulation-service's `GET /api/v1/loans?studentId=...&status=ACTIVE` over HTTP instead of
joining the `loans` table directly.

## Structure

```
cmd/api/                 → entry point (main.go)
internal/
├── domain/
│   ├── membership/        → Student aggregate, StudentRepository port, ActiveLoansChecker port
│   └── shared/             → Value Objects (Email) — duplicated per service, no shared Go module
├── application/usecase/  → CreateStudent (HU-02), UpdateStudent/DeactivateStudent/SearchStudents (HU-03)
├── config/                → environment variable loading
└── infrastructure/
    ├── http/               → chi router, middleware, handlers (primary adapters)
    ├── postgres/            → StudentRepository (secondary adapter)
    ├── circulation/          → HTTP client implementing ActiveLoansChecker
    └── logger/                → structured (zap) logger
```

No `migrations/` here — the schema, seeds, and migration tooling live in `lms-membership-db`
(`05-architecture/decisions/records/ADR-006-repo-per-context-decomposition.md`).

## Tech Stack

* **Language:** Go 1.25
* **Router:** chi
* **Database driver:** pgx (PostgreSQL, schema owned by `lms-membership-db`)

## Development

```bash
go mod download
go run ./cmd/api/...
```

Or, as part of the full assembled system (once `lms-infra` composes this repo alongside its
siblings — sibling-directory clone convention, `ADR-006`):

```bash
docker compose up --build membership-service
```

## Tests

```bash
go test ./...
```

## Migration scope

**Comes from** `lms-library` → `membership-service/`: `cmd/`, `internal/{domain,application,infrastructure}`,
`go.mod`, `go.sum`, `Dockerfile`, `Makefile`.

**Without `migrations/`** — the schema moves to `lms-membership-db`.

Keep the hexagonal layout already in place: domain with its ports and tests, use cases in
`application`, adapters in `infrastructure`.

The full map lives in `library-docs`.

---

## Branching

Three permanent branches. **None of them accepts a direct commit** — you enter through a child
branch and leave through a Pull Request.

```
develop  <--PR--  feat/... fix/... chore/...
qa       <--PR--  qa/...
main     <--PR--  release/...  hotfix/...
```

Promotion happens **by re-application** (`git cherry-pick -x`), never by merging one permanent
branch into another: `merge develop -> qa` and `merge qa -> main` do not exist in this model.

`main` requires **1 approval from `ariel5253`**. On `develop` and `qa` the team sets its own review
rule.

Full policy: `00-governance/branching-policy.md` in `library-docs`.

## Correlations

* Domain rules → `library-docs/02-domain/entities-and-rules.md`
* Database (own repo now) → [`lms-membership-db`](https://github.com/code-corhuila/lms-membership-db)
* API contract → `library-docs/07-api/contracts/openapi/membership-service.yaml`
* Migration map / status → `library-docs/09-microservices/repo-migration-map.md`
