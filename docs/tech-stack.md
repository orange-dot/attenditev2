# Tech Stack Definition

> Serbia Government Interoperability Platform

## Constraints

- **Deployment**: Kubernetes-native (all components must have Helm charts or operators)
- **Licensing**: OSS only (OSI-approved licenses)
- **Architecture**: Modular monolith (extract to microservices only when proven necessary)
- **Design**: DDD where complexity warrants it, simple CRUD elsewhere
- **Application Code**: No Java in application code - infrastructure components (Keycloak, Flink) are OK as black-box services
- **API Specifications**: OpenAPI 3.1 (REST) + AsyncAPI 3.0 (Events)
- **Philosophy**: No over-engineering - start simple, evolve based on real needs

---

## Confirmed Stack

### Infrastructure Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Container Orchestration | Kubernetes | Apache 2.0 | - |
| Ingress Controller | Traefik | Apache 2.0 | Native |
| Certificate Management | cert-manager | Apache 2.0 | Native |

> **Note:** Service mesh (Linkerd) removed. mTLS handled directly by services and Federation Layer. Distributed tracing via OpenTelemetry SDK → Tempo.

### Interoperability Layer (Custom)

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Agency Gateway | Custom (Go) | - | Helm charts |
| Trust Authority | Custom (Go) | - | Helm charts |
| API Gateway | Custom (Go) | - | Helm charts |
| API Spec (REST) | OpenAPI 3.1 | - | - |
| API Spec (Events) | AsyncAPI 3.0 | - | - |

> **Fully Custom Interoperability Layer:** Federation Layer (Trust Authority + Agency Gateway) for inter-agency communication. Custom API Gateway for external consumers (citizens, third parties). All built in Go with full control.

### Identity & Security Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Identity Provider | Keycloak | Apache 2.0 | Operator |
| Citizen Identity | Serbia eID (federated) | - | via Keycloak |
| Agency Identity | LDAP/AD (federated) | - | via Keycloak |
| Policy Engine | Open Policy Agent (OPA) | Apache 2.0 | Native |
| Access Model | RBAC + ABAC (XACML 3.0) | - | - |

> **Identity Federation:** Keycloak acts as identity broker. Citizens authenticate via Serbia eID ([eid.gov.rs](https://eid.gov.rs)), agency workers via their organization's LDAP/AD. Keycloak unifies sessions and maps attributes to platform roles.

### Messaging & Events Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Event Streaming | KurrentDB (EventStoreDB) | License | Helm charts |
| Task Queue | KurrentDB (EventStoreDB) | License | (same as above) |
| Secure Messaging | Custom (Go) | - | Helm charts |

> **KurrentDB (EventStoreDB)** provides event sourcing and streaming with built-in projections. Supports both gRPC and HTTP/AtomPub APIs for flexibility. **Custom Secure Messaging** replaces Matrix/Synapse - simpler E2EE messaging built on PostgreSQL + WebSocket.

### Workflow & Documents Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Workflow Engine | Temporal | MIT | Helm charts |
| Object Storage | MinIO | AGPL 3.0 | Operator |
| Document Metadata | PostgreSQL (same cluster) | PostgreSQL | CloudNativePG |
| Digital Signatures | PAdES/XAdES (eIDAS) | - | - |

> **Note:** Temporal uses code-first workflows (Go/TypeScript SDK) instead of BPMN diagrams. More flexible for developers, requires less visual tooling.

### Data Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Primary Database | PostgreSQL | PostgreSQL | CloudNativePG Operator |
| Full-Text Search | PostgreSQL (built-in FTS) | PostgreSQL | (same as above) |
| Cache | Valkey | BSD-3 | Operator |
| Stream Processing | Custom (Go) | - | Helm charts |

> **PostgreSQL FTS** replaces OpenSearch - uses GIN indexes, tsvector/tsquery for case search. Logs handled by Loki. **Custom Stream Processing** replaces Flink - Go workers consuming from KurrentDB for real-time analytics, SLA monitoring, and alerting.

### Observability Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Metrics | Prometheus | Apache 2.0 | Native |
| Dashboards | Grafana | AGPL 3.0 | Operator |
| Logs | Loki | AGPL 3.0 | Helm charts |
| Traces | Tempo | AGPL 3.0 | Helm charts |

### Application Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Backend Language | Go | BSD | - |
| Backend Framework | Chi | MIT | - |
| Frontend Framework | React + TypeScript | MIT | - |
| Mobile | React Native | MIT | - |

### DevOps Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| Source Control + CI/CD | GitLab CE (self-hosted) | MIT | Helm charts |
| GitOps | ArgoCD | Apache 2.0 | Native |
| Secret Management | OpenBao | MPL 2.0 | Helm charts |
| Secret Sync | External Secrets Operator | Apache 2.0 | Native |

### Emergency Services Layer

| Component | Technology | License | K8s Support |
|-----------|------------|---------|-------------|
| CAD System | Custom (Go) | - | Helm charts |
| Location Services | PostgreSQL + PostGIS | PostgreSQL | CloudNativePG |
| Real-time Tracking | KurrentDB + WebSocket | License | - |

> **Custom CAD** replaces Resgrid Core - built on Temporal (dispatch workflows), KurrentDB (real-time events), PostGIS (location/routing), WebSocket (live tracking). Tighter integration with platform.

---

## Architecture Approach

### Modular Monolith (Not Microservices)

| Aspect | Our Choice | Why |
|--------|------------|-----|
| Starting architecture | Modular monolith | Discover boundaries before splitting |
| Deployment | Single Go binary | Simple ops, easy debugging |
| Module communication | In-process interfaces | No network latency |
| Database | Shared PostgreSQL, separate schemas per module | Simple transactions |
| Extraction trigger | Proven need (scale, team, deploy cadence) | Not speculation |

### Why Not Microservices From Day One

```
MICROSERVICES PROBLEMS WE AVOID:
├── Distributed transactions (sagas complexity)
├── Network latency between services
├── Deployment coordination hell
├── Distributed debugging nightmare
├── Premature API contracts that lock in bad boundaries
└── Over-engineering before understanding the domain
```

### When to Extract a Module

Extract to separate service **only when you see**:

| Signal | Action |
|--------|--------|
| Module needs 10x more resources | Extract, scale independently |
| Different team owns it | Extract for autonomy |
| Different release cadence | Extract to deploy independently |
| Failure isolation critical | Extract to contain failures |
| Technology mismatch | Extract (rare in Go) |

**Default: Keep in monolith until proven otherwise.**

---

## Module Design

### DDD vs CRUD Decision

| Module | Approach | Rationale |
|--------|----------|-----------|
| **Case** | DDD | Complex lifecycle, business rules, events |
| **Dispatch** | DDD | State machines, real-time coordination |
| **Document** | DDD-lite | Lifecycle matters, simpler than Case |
| **Citizen** | CRUD | Reference data, no complex behavior |
| **Agency** | CRUD | Reference data, configuration |
| **Messaging** | Simple | Pass-through to infrastructure |
| **Audit** | Append-only | No domain logic, just logging |
| **Workflow** | Temporal | Delegated to Temporal engine |

### DDD Module Structure (Case, Dispatch)

```
/internal/case
├── /domain
│   ├── case.go           # Aggregate root
│   ├── case_event.go     # Domain events
│   ├── participant.go    # Entity
│   ├── status.go         # Value object
│   └── repository.go     # Interface (port)
├── /application
│   ├── commands.go       # CreateCase, TransferCase, etc.
│   ├── queries.go        # GetCase, ListCases, etc.
│   └── handlers.go       # Command/query handlers
├── /infrastructure
│   └── pg_repository.go  # PostgreSQL implementation
└── /api
    └── http.go           # Chi handlers
```

### CRUD Module Structure (Citizen, Agency)

```
/internal/citizen
├── model.go              # Simple struct
├── repository.go         # DB access
└── api.go                # HTTP handlers
```

### Module Rules

```
ENFORCED BY CODE STRUCTURE:
│
├── NO direct cross-module DB access
│   └── Module owns its tables, others use its API
│
├── NO circular dependencies
│   └── Dependency graph must be acyclic
│
├── Communication via interfaces
│   └── /shared/interfaces defines contracts
│
├── Events for cross-module notifications
│   └── Case emits "CaseCreated", Audit subscribes
│
└── Shared kernel is MINIMAL
    └── Only: IDs, Money, common errors
```

---

## Project Structure

```
/
├── cmd/
│   └── platform/
│       └── main.go                 # Single entry point
│
├── internal/                       # Private application code
│   │
│   ├── case/                       # Case module (DDD)
│   │   ├── domain/
│   │   │   ├── case.go             # Aggregate
│   │   │   ├── events.go           # Domain events
│   │   │   └── repository.go       # Port interface
│   │   ├── application/
│   │   │   ├── commands.go
│   │   │   ├── queries.go
│   │   │   └── service.go
│   │   ├── infrastructure/
│   │   │   └── postgres.go         # Adapter
│   │   └── api/
│   │       └── http.go
│   │
│   ├── citizen/                    # Citizen module (CRUD)
│   │   ├── model.go
│   │   ├── repository.go
│   │   └── api.go
│   │
│   ├── agency/                     # Agency module (CRUD)
│   ├── document/                   # Document module (DDD-lite)
│   ├── dispatch/                   # CAD module (DDD)
│   ├── messaging/                  # Secure messaging
│   ├── audit/                      # Audit logging
│   ├── workflow/                   # Temporal integration
│   │
│   ├── federation/                 # Federation Layer
│   │   ├── gateway/                # Agency Gateway
│   │   └── trust/                  # Trust Authority
│   │
│   └── shared/                     # Shared kernel (minimal!)
│       ├── types/
│       │   ├── id.go               # UUID wrapper
│       │   └── money.go            # Currency handling
│       ├── events/
│       │   └── bus.go              # Event bus interface
│       ├── auth/
│       │   └── middleware.go       # Keycloak integration
│       └── errors/
│           └── errors.go           # Common error types
│
├── pkg/                            # Public libraries (extractable)
│   ├── crypto/                     # Signing, certificates
│   └── kurrentdb/                  # KurrentDB client wrapper
│
├── api/                            # API specifications
│   ├── openapi/
│   │   └── platform.yaml           # OpenAPI 3.1 spec
│   └── asyncapi/
│       └── events.yaml             # AsyncAPI 3.0 spec
│
├── deploy/                         # Deployment configs
│   ├── helm/
│   │   └── platform/               # Helm chart
│   └── k8s/
│       └── base/                   # Kustomize base
│
├── scripts/                        # Dev scripts
├── docs/                           # Documentation
├── go.mod
└── go.sum
```

---

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                            KUBERNETES CLUSTER                             │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                           INGRESS                                   │  │
│  │       ┌──────────┐  ┌───────────┐  ┌────────────────┐              │  │
│  │       │  Traefik │  │cert-manager│  │ Custom API GW  │              │  │
│  │       └──────────┘  └───────────┘  └────────────────┘              │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                 FEDERATION LAYER (Separate Binaries)                │  │
│  │       ┌───────────────────┐       ┌───────────────────┐            │  │
│  │       │  TRUST AUTHORITY  │       │   AGENCY GATEWAY  │            │  │
│  │       │   (Single, HA)    │       │   (Per Agency)    │            │  │
│  │       └───────────────────┘       └───────────────────┘            │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │            PLATFORM - MODULAR MONOLITH (Single Go Binary)           │  │
│  │                                                                      │  │
│  │  ┌────────────────────────────────────────────────────────────┐    │  │
│  │  │                        MODULES                              │    │  │
│  │  │                                                             │    │  │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │    │  │
│  │  │  │  Case   │ │ Citizen │ │  Agency │ │Document │          │    │  │
│  │  │  │  (DDD)  │ │ (CRUD)  │ │ (CRUD)  │ │(DDD-lite)│          │    │  │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │    │  │
│  │  │                                                             │    │  │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐          │    │  │
│  │  │  │Dispatch │ │Messaging│ │  Audit  │ │Workflow │          │    │  │
│  │  │  │  (DDD)  │ │(Simple) │ │(Append) │ │(Temporal)│          │    │  │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘          │    │  │
│  │  │                                                             │    │  │
│  │  │  ┌─────────────────────────────────────────────────────┐  │    │  │
│  │  │  │              SHARED KERNEL (minimal)                 │  │    │  │
│  │  │  │         types │ events │ auth │ errors               │  │    │  │
│  │  │  └─────────────────────────────────────────────────────┘  │    │  │
│  │  └────────────────────────────────────────────────────────────┘    │  │
│  │                                                                      │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                  PLATFORM SERVICES (External)                       │  │
│  │                                                                      │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐               │  │
│  │  │ Keycloak │ │   OPA    │ │ OpenBao  │ │ KurrentDB│               │  │
│  │  │ Identity │ │  Policy  │ │ Secrets  │ │ (ESDB)   │               │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘               │  │
│  │                                                                      │  │
│  │  ┌──────────┐ ┌──────────┐                                          │  │
│  │  │ Temporal │ │  MinIO   │                                          │  │
│  │  │ Workflow │ │  Storage │                                          │  │
│  │  └──────────┘ └──────────┘                                          │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                          DATA LAYER                                 │  │
│  │  ┌───────────────────────────────────┐  ┌──────────┐               │  │
│  │  │           PostgreSQL              │  │  Valkey  │               │  │
│  │  │  • Schemas: case, citizen, agency │  │  Cache   │               │  │
│  │  │  • Full-Text Search (GIN/tsvector)│  │          │               │  │
│  │  │  • PostGIS (Location/Routing)     │  │          │               │  │
│  │  └───────────────────────────────────┘  └──────────┘               │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                    OBSERVABILITY & DEVOPS                           │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────┐ ┌──────────┐     │  │
│  │  │Prometheus│ │  Grafana │ │   Loki   │ │Tempo │ │  ArgoCD  │     │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────┘ └──────────┘     │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                          EXTERNAL / SELF-HOSTED                           │
│  ┌──────────────┐  ┌───────────────┐  ┌────────────────┐                 │
│  │  GitLab CE   │  │ React Web App │  │ React Native   │                 │
│  │  (CI/CD)     │  │   (Frontend)  │  │  Mobile Apps   │                 │
│  └──────────────┘  └───────────────┘  └────────────────┘                 │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## EU Compliance Checklist

- [ ] EIF alignment (legal, organizational, semantic, technical)
- [ ] eDelivery Access Point (Domibus)
- [ ] eID integration (eIDAS node)
- [ ] eSignature support (QES)
- [ ] OOTS readiness
- [ ] GDPR-equivalent data protection

---

## Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Primary DB | PostgreSQL over MongoDB | SSPL not OSS; stack compatibility |
| Workflow | Temporal over Camunda/Flowable | No Java app code; Go SDK |
| Documents | MinIO over Alfresco | Simpler; metadata in PostgreSQL |
| Secrets | OpenBao over HashiCorp Vault | BSL not OSS |
| Backend | Go + Chi | No Java; net/http compatible |
| Frontend | React + TypeScript | Largest ecosystem; talent pool |
| Mobile | React Native | Code sharing with React team |
| CI/CD | GitLab CE | Self-hosted; all-in-one |
| GitOps | ArgoCD | UI; multi-cluster ready |
| API Specs | OpenAPI 3.1 + AsyncAPI 3.0 | REST + Event-driven; code generation |
| **Service Mesh** | **None (removed Linkerd)** | Direct mTLS; Federation Layer handles inter-agency; reduce complexity |
| **Interoperability** | **Custom Go over X-Road** | Full Go stack; maximum control; tailored to needs |
| **API Gateway** | **Custom Go over Tyk** | Full control; simpler; integrated with stack |
| **Events/Queue** | **KurrentDB (EventStoreDB) over Kafka/RabbitMQ** | Event sourcing native; gRPC + HTTP APIs; projections built-in |
| **Messaging** | **Custom Go over Matrix/Synapse** | Simpler; E2EE on PostgreSQL + WebSocket |
| **Search** | **PostgreSQL FTS over OpenSearch** | Built-in; no Java; GIN indexes |
| **Stream Processing** | **Custom Go over Flink** | Simpler workers; KurrentDB consumers; full control |
| **CAD** | **Custom Go over Resgrid** | Tighter integration; Go stack; PostGIS |
| **Cache** | **Valkey over Redis** | Fully OSS (Linux Foundation fork) |
| **Citizen Identity** | **Serbia eID via Keycloak** | National eID federation; eIDAS compliant |
| **Agency Identity** | **LDAP/AD via Keycloak** | Federate existing agency directories |
| **Architecture** | **Modular Monolith over Microservices** | Discover boundaries first; simple ops; easy refactoring |
| **Design** | **DDD where complex, CRUD elsewhere** | No over-engineering; complexity only where needed |

## Notes

- **Maximum control philosophy**: Custom Go components where feasible, external only when truly necessary
- **Modular monolith**: Single deployable binary with well-defined module boundaries; extract to microservices only when proven necessary
- **DDD pragmatism**: Apply DDD patterns only to complex domains (Case, Dispatch); simple CRUD for reference data (Citizen, Agency)
- Only external infrastructure: Keycloak (identity critical), Temporal (excellent Go), MinIO (excellent Go), KurrentDB/EventStoreDB, PostgreSQL, Observability stack
- All custom components require security audit before production
- No Java in application code; Keycloak only Java infrastructure remaining
- OpenTelemetry SDK for distributed tracing (no sidecar needed)

---

## Custom Components Summary

### Separate Binaries (Federation Layer)

| Component | Purpose | Deployment |
|-----------|---------|------------|
| **Trust Authority** | Central PKI, member registry, service catalog | Single instance, HA |
| **Agency Gateway** | Per-agency mTLS, signing, audit | One per agency |
| **API Gateway** | External API management, rate limiting, auth | Single instance, HA |

### Modular Monolith (Platform Binary)

| Module | Approach | Purpose |
|--------|----------|---------|
| **Case** | DDD | Case lifecycle, cross-agency coordination |
| **Dispatch** | DDD | Emergency dispatch, unit tracking |
| **Document** | DDD-lite | Document lifecycle, signatures |
| **Citizen** | CRUD | Citizen reference data |
| **Agency** | CRUD | Agency configuration |
| **Messaging** | Simple | E2EE inter-agency chat |
| **Audit** | Append-only | Immutable audit logs |
| **Workflow** | Temporal | Long-running processes |

### Build Effort

| Component | Complexity | Notes |
|-----------|------------|-------|
| Trust Authority | High | Security-critical, needs audit |
| Agency Gateway | High | Security-critical, needs audit |
| API Gateway | Medium | Simpler than commercial gateways |
| Platform Monolith | High | But incremental - module by module |

---

## Federation Layer Specification

Custom Go-based interoperability layer replacing X-Road.

### Components

#### 1. Trust Authority (Central)

Single instance operated by national digital authority.

| Function | Description |
|----------|-------------|
| Member Registry | Agencies, their certificates, status |
| PKI / CA | Issue, renew, revoke agency certificates |
| Service Catalog | What services each agency exposes |
| Config Distribution | Push trust config to all gateways |
| Audit Log | All trust changes logged immutably |

#### 2. Agency Gateway (Per-Agency)

Each participating agency deploys their own gateway.

| Function | Description |
|----------|-------------|
| mTLS Termination | Verify peer agency certificates |
| Message Signing | Sign all outgoing requests (EdDSA) |
| Signature Verification | Verify all incoming signatures |
| Audit Logging | Every transaction logged (WORM) |
| Request Routing | Route to internal services |
| Rate Limiting | Protect against abuse |

### Message Protocol

```
┌─────────────────────────────────────────────────────────┐
│                    SignedEnvelope                        │
├─────────────────────────────────────────────────────────┤
│  id            : UUID                                    │
│  timestamp     : RFC3339 (signed)                        │
│  source        : AgencyID                                │
│  destination   : AgencyID                                │
│  service       : string (target service name)            │
│  correlation_id: UUID (for request-response tracking)    │
│  payload       : bytes (actual request/response)         │
│  signature     : bytes (EdDSA over all above fields)     │
│  cert_chain    : []bytes (for verification)              │
└─────────────────────────────────────────────────────────┘
```

### Security Requirements

| Requirement | Implementation |
|-------------|----------------|
| Transport | mTLS (TLS 1.3) between gateways |
| Signing | EdDSA (Ed25519) or RSA-PSS |
| Certificates | X.509v3, issued by Trust Authority CA |
| Audit Logs | Append-only, cryptographic hash chain |
| Key Storage | HSM recommended for production |

### Go Libraries

| Need | Library |
|------|---------|
| mTLS | `crypto/tls` (stdlib) |
| Certificates | `crypto/x509` (stdlib) |
| EdDSA | `crypto/ed25519` (stdlib) |
| HTTP/2 | `net/http` (stdlib) |
| Protobuf | `google.golang.org/protobuf` |
| JOSE | `github.com/go-jose/go-jose/v4` |

### Deployment Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                     NATIONAL TRUST AUTHORITY                         │
│                        (Central K8s Cluster)                         │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │  trust-authority (HA: 3 replicas)                            │    │
│  │  PostgreSQL (member registry, service catalog)               │    │
│  │  OpenBao (CA private keys)                                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                    Trust Config Distribution (gRPC stream)
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        ▼                          ▼                          ▼
┌───────────────┐          ┌───────────────┐          ┌───────────────┐
│ POLICE AGENCY │          │  HEALTHCARE   │          │ SOCIAL SVCS   │
│   Gateway     │◄────────►│   Gateway     │◄────────►│   Gateway     │
│               │  mTLS    │               │  mTLS    │               │
└───────┬───────┘          └───────┬───────┘          └───────┬───────┘
        │                          │                          │
        ▼                          ▼                          ▼
┌───────────────┐          ┌───────────────┐          ┌───────────────┐
│ Internal      │          │ Internal      │          │ Internal      │
│ Services      │          │ Services      │          │ Services      │
└───────────────┘          └───────────────┘          └───────────────┘
```

### EU Compliance Considerations

| Requirement | Approach |
|-------------|----------|
| eDelivery | Implement AS4 profile for EU cross-border (future) |
| eIDAS | Support QES via external signing service |
| EIF | Document alignment with 4 interoperability layers |
| Audit | Retain logs per GDPR (7+ years for gov) |

---

## Serbian eID Integration

Integration with Serbia's national electronic identification system ([eid.gov.rs](https://eid.gov.rs)).

### Serbia eID Overview

| Aspect | Details |
|--------|---------|
| Portal | [eid.gov.rs](https://eid.gov.rs/en-US/start) |
| Mobile App | ConsentID |
| eSignature | Qualified certificates in cloud |
| SSO | Single Sign-On across government portals |
| Compliance | eIDAS-compliant (since 2017) |
| Digital Wallet | Coming end of 2025 |

### Authentication Levels

| Method | Assurance Level | Use Case |
|--------|-----------------|----------|
| Username + Password | Basic | Low-risk services |
| ConsentID Mobile | High | Standard government services |
| Qualified Certificate | Highest | Legal documents, signatures |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         IDENTITY FEDERATION                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│                         ┌──────────────────┐                            │
│                         │     KEYCLOAK     │                            │
│                         │  (Identity Hub)  │                            │
│                         └────────┬─────────┘                            │
│                                  │                                       │
│              ┌───────────────────┼───────────────────┐                  │
│              │                   │                   │                  │
│              ▼                   ▼                   ▼                  │
│   ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐       │
│   │   SERBIA eID     │ │   AGENCY LDAP    │ │  LOCAL ACCOUNTS  │       │
│   │   (Citizens)     │ │   (Workers)      │ │  (Admins/Test)   │       │
│   │                  │ │                  │ │                  │       │
│   │ • ConsentID      │ │ • Police AD      │ │ • System admins  │       │
│   │ • QES Cert       │ │ • Health LDAP    │ │ • Service accts  │       │
│   │ • Username/Pass  │ │ • Social LDAP    │ │                  │       │
│   └──────────────────┘ └──────────────────┘ └──────────────────┘       │
│                                                                          │
│   Protocol: SAML 2.0 or OIDC (TBD - depends on Serbia eID specs)        │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Integration Steps

| Step | Action | Status |
|------|--------|--------|
| 1 | Contact Office for IT and eGovernment | ⏳ Required |
| 2 | Apply for Service Provider registration | ⏳ Required |
| 3 | Receive technical specifications (SAML/OIDC) | ⏳ Waiting |
| 4 | Configure Keycloak identity broker | ⏳ Pending specs |
| 5 | Map Serbia eID attributes to platform roles | ⏳ Pending specs |
| 6 | Test in sandbox environment | ⏳ Pending access |
| 7 | Production deployment | ⏳ Pending approval |

### Keycloak Configuration (Template)

```yaml
# keycloak/realms/serbia-gov.yaml
realm: serbia-gov-platform

identityProviders:
  # Serbia eID - Citizens
  - alias: serbia-eid
    providerId: saml  # or oidc - TBD
    enabled: true
    trustEmail: true
    firstBrokerLoginFlowAlias: first-broker-login
    config:
      # To be filled when specs received from eid.gov.rs
      entityId: "https://platform.gov.rs/auth"
      singleSignOnServiceUrl: "https://eid.gov.rs/saml/sso"  # placeholder
      singleLogoutServiceUrl: "https://eid.gov.rs/saml/slo"  # placeholder
      nameIDPolicyFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
      principalType: "SUBJECT"
      signatureAlgorithm: "RSA_SHA256"
      wantAuthnRequestsSigned: true
      validateSignature: true
      # signingCertificate: "..." # from Serbia eID metadata

  # Agency LDAP - Example for Police
  - alias: police-ldap
    providerId: ldap
    enabled: true
    config:
      vendor: "ad"  # or "other" for OpenLDAP
      connectionUrl: "ldaps://ldap.mup.gov.rs"
      usersDn: "ou=users,dc=mup,dc=gov,dc=rs"
      bindDn: "cn=keycloak,ou=service,dc=mup,dc=gov,dc=rs"
      bindCredential: "${POLICE_LDAP_PASSWORD}"
      userObjectClasses: "person, organizationalPerson, user"
      usernameLDAPAttribute: "sAMAccountName"
      uuidLDAPAttribute: "objectGUID"

# Attribute mappers - Serbia eID to Platform
identityProviderMappers:
  - name: serbia-eid-jmbg
    identityProviderAlias: serbia-eid
    identityProviderMapper: saml-user-attribute-idp-mapper
    config:
      syncMode: INHERIT
      user.attribute: jmbg  # Serbian personal ID number
      attribute.name: "urn:oid:1.3.6.1.4.1.XXXXX.1"  # placeholder OID

  - name: serbia-eid-name
    identityProviderAlias: serbia-eid
    identityProviderMapper: saml-user-attribute-idp-mapper
    config:
      syncMode: INHERIT
      user.attribute: fullName
      attribute.name: "urn:oid:2.5.4.3"  # commonName
```

### User Attribute Mapping

| Serbia eID Attribute | Platform Attribute | Purpose |
|----------------------|-------------------|---------|
| JMBG (Personal ID) | `citizen_id` | Unique citizen identifier |
| Full Name | `display_name` | Display purposes |
| Date of Birth | `date_of_birth` | Age verification |
| Address | `registered_address` | Service eligibility |
| eID Level | `assurance_level` | Access control decisions |

### Security Considerations

| Concern | Mitigation |
|---------|------------|
| Session hijacking | Short session timeouts, re-auth for sensitive ops |
| Token replay | Nonce validation, timestamp checks |
| Man-in-middle | TLS 1.3 only, certificate pinning |
| Account linking | Require high assurance for linking existing accounts |
| Consent | Explicit consent UI before first login |

### Contacts

| Organization | Purpose | Website |
|--------------|---------|---------|
| Office for IT and eGovernment | eID integration approval | [ite.gov.rs](https://www.ite.gov.rs/) |
| Ministry of Interior | ID card certificates | [mup.gov.rs](https://www.mup.gov.rs/) |
| eID Portal | User registration | [eid.gov.rs](https://eid.gov.rs/) |
