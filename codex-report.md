# Codex Report - attenditev2

## Scope
- Full repo scan with emphasis on implemented vs mock/partial features.
- Special focus: audit trail and chain verification.

## Repo Map (high level)
- Go backend in `cmd/` + `internal/` with module-based design.
- React frontend in `web/` (Vite + Tailwind).
- Python AI mock service in `services/ai-mock`.
- Infra and deployment in `deploy/` (Docker, OPA, monitoring).
- Docs in `docs/` (architecture, domain, events, security).
- `api/` and `mobile/` directories are present but empty.

## What is Implemented (by area)
### Backend (Go)
- Platform bootstrapping, config, DB init, and KurrentDB event bus (conditional on availability). `cmd/platform/main.go`, `internal/shared/config/config.go`, `internal/shared/events/bus.go`
- Agency CRUD with Postgres repository + HTTP API. `internal/agency/*`
- Case domain model (DDD), Postgres repository, and HTTP API. `internal/case/*`
- Document module with versioning, signatures (data model), and Postgres repository + HTTP API. `internal/document/*`
- Audit module with hash chain and verification logic; KurrentDB repository + HTTP API + subscriber. `internal/audit/*`
- Federation trust authority (in-memory) with certificate issuance. `internal/federation/trust/*`
- KurrentDB client/abstractions. `internal/kurrentdb/*`
- Privacy guard middleware + pseudonymization primitives. `internal/privacy/*`
- Coordination service (event processing, escalation, protocols) and notification service core. `internal/coordination/*`, `internal/notification/*`
- Health and social adapters with real integration code (SQL Server + mTLS HTTP client). `internal/adapters/health/*`, `internal/adapters/social/*`
- TSA library (RFC3161) + multi-agency witness implementation (library level). `internal/tsa/*`

### Frontend (React)
- Multi-page UI scaffold with pages for audit, chain security, cases, documents, federation, privacy, AI, etc. `web/src/pages/*`
- Audit and chain security demo UI that calls audit verify/checkpoints endpoints. `web/src/pages/ChainSecurity.tsx`

### Infra/DevOps
- Docker compose for dev/prod/demo; Keycloak, OPA, KurrentDB, Postgres, Grafana/Loki/Promtail. `deploy/docker/*`, `docker-compose.yml`
- OPA policies defined (case, document, admin, dispatch). `deploy/opa/policies/*`
- DB migrations for identity/cases/documents/audit. `internal/shared/database/migrations/*`

## Mock / Partial / Not Wired (key gaps)
### Explicit mocks
- AI service is a mock Python service; Go AI module just proxies to it. `services/ai-mock/*`, `internal/ai/*`, `cmd/platform/main.go`
- Notification providers are mock (push/SMS/email) and only print output. `internal/notification/providers.go`

### Partial or not wired
- Audit storage has **two** implementations, but only KurrentDB is wired in main:
  - KurrentDB repo is used when event bus is up. `cmd/platform/main.go`, `internal/audit/kurrentdb_repository.go`
  - Postgres audit repository exists but does not implement full `AuditRepository` interface and is not used. `internal/audit/repository.go`, `internal/audit/interface.go`
- TSA + witness configuration exists, but `cmd/platform/main.go` always uses `NewLocalWitness()` (no RFC3161 or multi-agency wiring). `internal/audit/api.go`, `internal/audit/checkpoint.go`, `internal/shared/config/config.go`
- Federation gateway is implemented but not mounted in the main router. Trust Authority is in-memory only (nil repo). `internal/federation/gateway/*`, `cmd/platform/main.go`, `internal/federation/trust/authority.go`
- Document signatures are modeled and stored, but signing uses empty signature/cert/token data; no signature verification flow. `internal/document/model.go`, `internal/document/api.go`
- The trust registry lists `/api/v1/documents/verify` as a service, but no such endpoint exists in the document API. `cmd/platform/main.go`, `internal/document/api.go`
- Privacy depseudonymization and AI access controllers define interfaces but have no persistence implementations. `internal/privacy/depseudonymization.go`, `internal/privacy/ai_access.go`
- Pseudonymization decryption is explicitly not implemented (HSM/encrypted storage placeholder). `internal/privacy/pseudonymization.go`
- OPA policy client/middleware exists but is not attached in `cmd/platform/main.go`. `internal/shared/policy/*`
- Coordination and notification services are not wired into the main server; they are library-level services. `internal/coordination/*`, `internal/notification/*`, `cmd/platform/main.go`
- `api/` and `mobile/` directories are empty placeholders.
- No tests found in repo.

## Audit and Chain Verification (deep dive)
### Current runtime behavior
- Audit module is only mounted if KurrentDB event bus is available. `cmd/platform/main.go`
- Events from case/document/agency/auth/simulation are subscribed to and appended to the audit stream. `internal/audit/subscriber.go`
- Hash chain uses canonical JSON with sorted keys for deterministic hashing. `internal/audit/model.go`
- Chain verification checks content hash integrity and prev_hash linkage. `internal/audit/kurrentdb_repository.go`

### Checkpoints and witnesses
- Checkpoints are stored in a dedicated KurrentDB stream. `internal/audit/kurrentdb_repository.go`
- Default witness is local and always "confirms" (no external proof). `internal/audit/checkpoint.go`
- RFC3161 TSA server and multi-agency witness are implemented but not configured in runtime. `internal/tsa/*`, `internal/audit/checkpoint.go`, `internal/shared/config/config.go`

### Postgres audit path (not wired)
- DB schema has append-only triggers for `audit.entries`, but the platform uses KurrentDB for audit in main. `internal/shared/database/migrations/001_initial_schemas.sql`, `cmd/platform/main.go`
- Postgres repository also has VerifyChain logic, but misses checkpoint/witness methods required by the interface. `internal/audit/repository.go`, `internal/audit/interface.go`

### Frontend alignment
- Chain security UI expects `/api/v1/audit/verify` and `/api/v1/audit/checkpoints` and can run live demos. `web/src/pages/ChainSecurity.tsx`
- This works only when KurrentDB is available and audit module is mounted.

## Additional Observations / Risks
- README mentions NATS, but implementation uses KurrentDB for event bus. `README.md`, `internal/shared/events/bus.go`
- Trust Authority certificates are self-signed in memory; no persistence or revocation list storage. `internal/federation/trust/authority.go`
- Document signing has no crypto validation path; signatures are effectively placeholders. `internal/document/*`
- Privacy guard can log violations to audit if KurrentDB is up; otherwise it silently no-ops. `cmd/platform/main.go`
- No automated tests; add coverage before pilot rollout.

## Suggested Next Steps (if you want to close gaps)
1. Wire TSA config to audit checkpoint witness selection (local/RFC3161/multi-agency).
2. Add persistence layer for Trust Authority, depseudonymization, and AI access requests.
3. Implement document verification endpoint + real signature validation/TSA tokens.
4. Integrate OPA middleware into API routes and validate policies in runtime.
5. Add tests for audit chain verification and checkpoint flows.
