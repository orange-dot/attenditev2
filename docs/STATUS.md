# Project Status

> Last updated: 2026-01-10

## Current Phase: Phase 7 - Pilot Deployment (Ready)

MVP core platform complete (Phases 1-6). Adapters and coordination service implemented. Ready for pilot deployment.

**Next milestone:** Deploy to Kikinda pilot (OB + CSR + DZ)

---

## Implementation Progress

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1: Foundation | **Complete** | Go project bootstrap, config, database, auth middleware, event bus |
| Phase 2: Core Modules | **Complete** | Agency, Case, Document, Audit modules with event publishing |
| Phase 3: Federation | **Complete** | Trust Authority, Agency Gateway, Cross-agency communication |
| Phase 4: Authorization | **Complete** | OPA integration, Prometheus metrics, security hardening |
| Phase 5: Legacy Adapters | **Complete** | Heliant HIS, Socijalna karta adapters, FHIR transformation |
| Phase 6: Real-time Coordination | **Complete** | Coordination service, notification service, protocol engine |
| Phase 7: Pilot Deployment | **Ready** | Kikinda (OB + CSR + DZ), K8s deployment, training |

---

## Completed Components

### Phase 1: Foundation
- [x] Go module initialization (`github.com/serbia-gov/platform`)
- [x] Folder structure per tech-stack.md
- [x] Main application entry point
- [x] Configuration management with env vars
- [x] PostgreSQL connection pool with migrations
- [x] JWT authentication middleware
- [x] NATS event bus abstraction
- [x] Shared types (ID, JMBG, Address)
- [x] Error handling package

### Phase 2: Core Modules
- [x] **Agency Module** - CRUD operations for agencies and workers
- [x] **Case Module (DDD)** - Full aggregate with lifecycle, participants, assignments, events
- [x] **Document Module** - Documents with versioning, signatures, sharing
- [x] **Audit Module** - Append-only logging with hash chain integrity
- [x] **Event Publishing** - All modules publish events to NATS
- [x] **Audit Subscriber** - Automatic audit logging from domain events

### Phase 3: Federation Layer
- [x] **Trust Authority** - Agency registry and certificate management
- [x] **Agency Gateway** - Secure cross-agency communication with EdDSA signatures

### Phase 4: Authorization & Observability
- [x] **OPA Policy Client** - Integration with Open Policy Agent
- [x] **Authorization Policies** - Rego policies for case and document access
- [x] **Policy Middleware** - HTTP middleware for policy enforcement
- [x] **Prometheus Metrics** - HTTP, business, and database metrics
- [x] **Security Headers** - X-Frame-Options, CSP, XSS protection
- [x] **Rate Limiting** - Global and per-IP rate limiting
- [x] **Request Logging** - Structured request logging
- [x] **Input Sanitization** - Request body size limits
- [x] **CORS Configuration** - Configurable CORS middleware

### Phase 5: Legacy System Adapters
- [x] **Health Adapter Interface** - Unified interface for HIS systems
- [x] **Health Types** - PatientRecord, Hospitalization, LabResult, Prescription, Diagnosis
- [x] **Heliant HIS Adapter** - SQL Server-based adapter for Heliant HIS (OB Kikinda)
- [x] **FHIR R4 Transformation** - Patient and Encounter FHIR resources with Serbian profiles
- [x] **Social Adapter Interface** - Unified interface for social protection systems
- [x] **Social Types** - BeneficiaryStatus, FamilyUnit, SocialCase, RiskAssessment
- [x] **Socijalna Karta Client** - mTLS-secured API client for Servisna magistrala

### Phase 6: Real-time Coordination
- [x] **Coordination Service** - Event processing with worker pool
- [x] **Enrichment Service** - Cross-system context aggregation (health + social)
- [x] **Protocol Engine** - Configurable coordination protocols
- [x] **Default Protocols** - Hospital admission/discharge, child protection, domestic violence
- [x] **Escalation Service** - Timeout-based escalation with multiple levels
- [x] **Notification Service** - Multi-channel notification delivery
- [x] **Notification Providers** - Push, SMS, Email with mock implementations

---

## API Endpoints

### Core Modules (`/api/v1`)

| Module | Endpoints |
|--------|-----------|
| Agency | CRUD for agencies and workers |
| Cases | Create, update, lifecycle, participants, assignments, sharing, transfer |
| Documents | CRUD, versions, signatures, sharing, archive, void |
| Audit | List, get, verify chain, by resource (admin only) |

### Federation (`/api/v1/federation`)

| Component | Endpoints |
|-----------|-----------|
| Trust Authority | Agency registry, services, certificates |
| Gateway | Send/receive cross-agency requests |

### Observability

| Endpoint | Description |
|----------|-------------|
| `/health` | Basic health check |
| `/ready` | Readiness with dependency checks |
| `/metrics` | Prometheus metrics |

---

## Security Features

- **Authentication**: JWT with Keycloak support
- **Authorization**: OPA policy engine with Rego policies
- **Rate Limiting**: Per-IP and global rate limits
- **Security Headers**: CSP, X-Frame-Options, XSS protection
- **Input Validation**: Request body size limits
- **Audit Trail**: Immutable append-only log with hash chain
- **Federation Security**: EdDSA signatures, certificate verification

---

## Metrics Exposed

### HTTP Metrics
- `http_requests_total` - Request count by method/path/status
- `http_request_duration_seconds` - Request latency histogram
- `http_requests_in_flight` - Current active requests

### Business Metrics
- `cases_created_total` - Case creation count
- `cases_status_changed_total` - Status transitions
- `documents_created_total` - Document creation count
- `documents_signed_total` - Signatures count
- `federation_requests_total` - Cross-agency requests
- `audit_entries_total` - Audit entries created
- `authorization_decisions_total` - Allow/deny decisions

### Infrastructure Metrics
- `db_connections_active` - Active DB connections
- `db_query_duration_seconds` - Query latency

---

## OPA Policies

### Case Access Policy (`platform/case`)
- Owner agency access
- Shared agency access (by access level)
- Platform/agency admin override
- Worker assignment access
- Field-level access (JMBG, audit trail)

### Document Access Policy (`platform/document`)
- Owner agency access
- Shared agency access
- Creator access
- Signer access for pending signatures
- Archive/void protection

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP port |
| `ENV` | development | Environment mode |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | platform | Database user |
| `DB_PASSWORD` | platform | Database password |
| `DB_NAME` | platform | Database name |
| `NATS_URL` | nats://localhost:4222 | NATS server |
| `JWT_SECRET` | dev-secret | JWT signing key |
| `OPA_URL` | http://localhost:8181 | OPA server |
| `OPA_ENABLED` | false | Enable OPA |

---

## Running the Application

```bash
# Start infrastructure
docker-compose up -d

# Build and run
go build -o bin/platform ./cmd/platform
./bin/platform

# With OPA enabled
OPA_ENABLED=true ./bin/platform
```

---

## Project Structure

```
cmd/
  platform/           # Main application
internal/
  adapters/           # Legacy system adapters
    health/           # Health system adapter interface
      fhir/           # FHIR R4 transformation
      heliant/        # Heliant HIS adapter
    social/           # Social system adapter interface
      socialcard/     # Socijalna Karta API client
  agency/             # Agency CRUD module
  audit/              # Audit logging + subscriber
  case/
    api/              # HTTP handlers
    domain/           # Domain model (DDD)
    infrastructure/   # PostgreSQL repository
  coordination/       # Real-time coordination service
  document/           # Document management
  federation/
    trust/            # Trust Authority
    gateway/          # Agency Gateway
  notification/       # Notification service
  shared/
    auth/             # JWT middleware
    config/           # Configuration
    database/         # PostgreSQL + migrations
    errors/           # Error types
    events/           # NATS event bus
    metrics/          # Prometheus metrics
    middleware/       # Security middleware
    policy/           # OPA client
    types/            # Shared types
deploy/
  opa/
    policies/         # Rego policies
docs/                 # Documentation
```

---

## Next Steps

### Phase 7: Pilot Deployment (Kikinda)
1. Deploy Heliant adapter to OB Kikinda server
2. Configure Socijalna Karta API access via Servisna magistrala
3. Deploy coordination service to Kragujevac data center
4. Set up K8s manifests for pilot environment
5. Configure test agencies (OB Kikinda, CSR Kikinda, DZ Kikinda)
6. End-to-end testing with real data
7. User training documentation
8. Go-live with monitoring

### Future Phases
1. GIZ adapter for residential care facilities
2. InfoMedis adapter (alternative HIS systems)
3. eZdravlje integration via RFZO
4. Mobile application for field workers
5. Expand to additional municipalities

---

## Architecture Documents

| Document | Description |
|----------|-------------|
| `docs/tech-stack.md` | Technology decisions |
| `docs/domain-model.md` | Domain entities |
| `docs/event-catalog.md` | Event types |
| `docs/security-model.md` | Security policies |
| `docs/MVP-IMPLEMENTATION-PLAN.md` | Full implementation plan (v2.0) |
| `docs/adapter-architecture.md` | Adapter architecture for legacy system integration |
| `docs/it-sistem-srb-research-init.md` | Research on Serbian IT systems fragmentation |
| `docs/outreach-letters.md` | Outreach letters to institutions |

---

## Resume Command

```
Continue with Phase 7 - Pilot Deployment

All adapters and coordination services implemented:
- internal/adapters/health/ - Health adapter interface + Heliant HIS
- internal/adapters/health/fhir/ - FHIR R4 transformation
- internal/adapters/social/ - Social adapter interface + Socijalna Karta
- internal/coordination/ - Real-time coordination service
- internal/notification/ - Notification service

Next: Deploy to Kikinda pilot (OB + CSR + DZ)

See: docs/MVP-IMPLEMENTATION-PLAN.md (v2.0) for detailed plan
```
