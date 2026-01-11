# MVP Implementation Plan
## Serbia Government Interoperability Platform

> Verzija: 2.0 | Datum: 2026-01-10

---

## Executive Summary

Ovaj dokument definiše plan implementacije za Minimum Viable Product (MVP) Platforme za Interoperabilnost Vlade Srbije. MVP će demonstrirati ključne sposobnosti sistema kroz funkcionalan prototip koji povezuje pilot institucije u Kikindi: **Opštu bolnicu, Centar za socijalni rad, i Dom zdravlja**.

**Cilj MVP-a:** Dokazati tehničku izvodljivost i operativnu vrednost platforme kroz realan scenario koordinacije zdravstvenih i socijalnih službi - problem koji trenutno ne rešava nijedan postojeći sistem u Srbiji.

### Ključni problem koji rešavamo

```
DANAS (bez platforme):                    SA PLATFORMOM:

Policija primi poziv u 2 ujutru           Adapter detektuje hospitalizaciju
    │                                          │
    ▼                                          ▼
Zove CSR na mobilni                       Event: case.emergency
    │                                          │
    ▼                                          ▼
Niko ne odgovara                          Automatska notifikacija CSR
    │                                          │
    ▼                                          ▼
Čeka do jutra                             ACK u roku od 5 minuta
    │                                          │
    ▼                                          ▼
Žrtva bez pomoći 8+ sati                  Koordinirana intervencija < 30 min
```

---

## Kontekst: Zašto postojeći sistemi nisu rešenje

### Paradoks srpske eUprave

Srbija je rangirana **2. u Evropi** na World Bank GovTech Maturity Index, ali:

| Sistem | Investicija | Problem |
|--------|-------------|---------|
| **SOZIS** (171 CSR) | €12.09M | Usporava rad, nema integraciju sa zdravstvom |
| **Socijalna karta** | €5.6M | 44,000 izgubilo pomoć, algoritam tajna |
| **112 hitni broj** | €27M+ | **NIJE OPERATIVAN** od 2019 |
| **eZUP** | 400+ institucija | Samo dokumenti, ne real-time |

### Šta nedostaje

1. **Real-time koordinacija** - eZUP je query-based, ne event-driven
2. **Health-social integracija** - IZIS i SOZIS ne komuniciraju
3. **24/7 socijalna hitna** - Ne postoji nacionalno
4. **Rezidencijalna nega** - GIZ sistem potpuno izolovan

---

## MVP Scope Definition

### Included in MVP (U opsegu)

| Komponenta | Funkcionalnost | Prioritet | Status |
|------------|----------------|-----------|--------|
| **Identity** | Keycloak + simuliran eID login | P0 | ✅ Done |
| **Agency Module** | CRUD za agencije i radnike | P0 | ✅ Done |
| **Case Module** | Kreiranje, ažuriranje, deljenje predmeta | P0 | ✅ Done |
| **Document Module** | Upload, pregled, verzioniranje, potpisi | P0 | ✅ Done |
| **Audit** | Append-only logging sa hash chain | P0 | ✅ Done |
| **Federation** | Trust Authority + Agency Gateway | P0 | ✅ Done |
| **Authorization** | OPA policies, security middleware | P0 | ✅ Done |
| **Metrics** | Prometheus + Grafana | P0 | ✅ Done |
| **Health Adapter** | Heliant HIS integration | P0 | 🔄 New |
| **Social Adapter** | SOZIS/Soc.karta integration | P0 | 🔄 New |
| **Real-time Coordination** | Emergency protocol | P0 | 🔄 New |
| **Web UI** | Admin panel + radna konzola | P1 | Pending |

### Excluded from MVP (Van opsega za sada)

| Komponenta | Razlog | Post-MVP faza |
|------------|--------|---------------|
| Dispatch/CAD | Zavisi od 112 implementacije | Phase 7 |
| Mobile Apps | Fokus na web | Phase 8 |
| Qualified Signatures | eIDAS QES integracija | Phase 6 |
| Full eID Integration | Registracija kod eid.gov.rs | Phase 6 |
| Workflow Engine | Temporal | Phase 7 |
| GIZ Adapter | Rezidencijalne ustanove | Phase 6 |

---

## Implementation Phases

### Phase 1: Foundation (Temelj) ✅ COMPLETE

```
DELIVERABLES:
├── Go project bootstrap
│   ├── Folder structure per tech-stack.md          ✅
│   ├── Chi router setup                            ✅
│   ├── Configuration management                    ✅
│   └── Health checks                               ✅
│
├── Database setup
│   ├── PostgreSQL schemas                          ✅
│   ├── Initial migrations                          ✅
│   └── Connection pooling                          ✅
│
├── Authentication
│   ├── JWT validation middleware                   ✅
│   └── Basic RBAC (admin, worker roles)            ✅
│
└── Event infrastructure
    ├── NATS JetStream abstraction                  ✅
    ├── Event bus                                   ✅
    └── Audit event subscriber                      ✅
```

---

### Phase 2: Core Modules ✅ COMPLETE

```
DELIVERABLES:
├── Agency Module (CRUD)                            ✅
│   ├── Agency entity + API
│   ├── Worker entity + API
│   └── Event publishing
│
├── Case Module (DDD)                               ✅
│   ├── Case aggregate
│   ├── Case lifecycle (draft → open → closed)
│   ├── Participant management
│   ├── Assignment management
│   ├── Cross-agency sharing (access levels)
│   └── Event publishing
│
├── Document Module                                 ✅
│   ├── Document entity with versions
│   ├── Signature support
│   ├── Sharing between agencies
│   ├── Archive/Void operations
│   └── Event publishing
│
└── Audit Module                                    ✅
    ├── Append-only log table
    ├── Hash chain integrity
    └── Audit subscriber for events
```

---

### Phase 3: Federation Layer ✅ COMPLETE

```
DELIVERABLES:
├── Trust Authority                                 ✅
│   ├── Agency registry
│   ├── Service catalog
│   └── Certificate management (Ed25519)
│
├── Agency Gateway                                  ✅
│   ├── Request signing (EdDSA)
│   ├── Signature verification
│   └── Cross-agency message relay
│
└── Cross-Agency Operations                         ✅
    ├── Share case with another agency
    ├── View shared cases
    └── Transfer case ownership
```

---

### Phase 4: Authorization & Observability ✅ COMPLETE

```
DELIVERABLES:
├── OPA Integration                                 ✅
│   ├── Policy engine client
│   ├── Case access policies (Rego)
│   ├── Document access policies (Rego)
│   └── Authorization middleware
│
├── Observability                                   ✅
│   ├── Prometheus metrics
│   │   ├── HTTP metrics
│   │   ├── Business metrics
│   │   └── Database metrics
│   └── /metrics endpoint
│
└── Security Hardening                              ✅
    ├── Security headers (CSP, X-Frame, etc.)
    ├── Rate limiting (global + per-IP)
    ├── CORS configuration
    └── Input sanitization
```

---

### Phase 5: Legacy System Adapters 🔄 NEW

**Trajanje: ~6 nedelja**

Ova faza dodaje adaptere za integraciju sa postojećim sistemima. Adapteri se izvršavaju na edge lokacijama (bolnice, CSR) i sinhronizuju sa centralnom platformom.

```
DELIVERABLES:
├── Health Adapter Framework
│   ├── internal/adapters/health/adapter.go         Interface definition
│   ├── internal/adapters/health/types.go           Common health types
│   └── internal/adapters/health/fhir/              FHIR transformation
│
├── Heliant HIS Adapter (Pilot: OB Kikinda)
│   ├── internal/adapters/health/heliant/
│   │   ├── adapter.go                              Main adapter
│   │   ├── queries.go                              SQL queries
│   │   ├── mapper.go                               Data mapping
│   │   └── poller.go                               Real-time polling
│   │
│   ├── Capabilities:
│   │   ├── FetchPatientRecord(jmbg)                Patient data
│   │   ├── FetchHospitalizations(jmbg, range)      Hospital stays
│   │   ├── FetchLabResults(jmbg, range)            Lab results
│   │   ├── FetchPrescriptions(jmbg)                Active prescriptions
│   │   ├── SubscribeAdmissions()                   Real-time admissions
│   │   └── SubscribeDischarges()                   Real-time discharges
│   │
│   └── Output: FHIR R4 resources (Patient, Encounter, Observation)
│
├── Social Protection Adapter Framework
│   ├── internal/adapters/social/adapter.go         Interface definition
│   └── internal/adapters/social/types.go           Common social types
│
├── Socijalna Karta Client (via Servisna magistrala)
│   ├── internal/adapters/social/socialcard/
│   │   ├── client.go                               API client (mTLS)
│   │   └── mapper.go                               Data mapping
│   │
│   └── Capabilities:
│       ├── FetchBeneficiaryStatus(jmbg)            Social assistance status
│       ├── FetchFamilyComposition(jmbg)            Family unit
│       └── FetchPropertyData(jmbg)                 Property records
│
├── SOZIS Adapter (Pilot: CSR Kikinda)
│   ├── internal/adapters/social/sozis/
│   │   ├── adapter.go                              Main adapter
│   │   └── poller.go                               Case updates poller
│   │
│   └── Capabilities:
│       ├── FetchOpenCases(jmbg)                    Active CSR cases
│       ├── FetchCaseHistory(jmbg)                  Historical cases
│       ├── FetchRiskAssessment(jmbg)               Risk evaluation
│       └── SubscribeCaseUpdates()                  Real-time updates
│
└── Edge Deployment Package
    ├── deploy/edge/
    │   ├── Dockerfile                              Adapter container
    │   ├── docker-compose.yml                      Local deployment
    │   └── config/                                 Per-site configuration
    │
    └── Features:
        ├── Local cache (SQLite/BoltDB)             Offline resilience
        ├── Secure tunnel (mTLS)                    Encrypted sync
        ├── Retry with backoff                      Network resilience
        └── Conflict resolution                     Data consistency
```

**Verifikacija:** Adapter na OB Kikinda može preuzeti podatke pacijenta i poslati ih platformi. CSR Kikinda vidi hospitalizacije svojih korisnika.

---

### Phase 6: Real-time Coordination 🔄 NEW

**Trajanje: ~4 nedelje**

Ova faza implementira event-driven koordinaciju između agencija - ključna funkcionalnost koja ne postoji u eZUP-u.

```
DELIVERABLES:
├── Coordination Service
│   ├── internal/coordination/
│   │   ├── service.go                              Main service
│   │   ├── protocol.go                             Event types
│   │   ├── enrichment.go                           Context enrichment
│   │   └── escalation.go                           Timeout handling
│   │
│   └── Capabilities:
│       ├── Receive events from adapters
│       ├── Enrich with cross-system context
│       ├── Route to relevant agencies
│       └── Track acknowledgments
│
├── Emergency Protocol
│   ├── Event Types:
│   │   ├── case.emergency                          Critical priority
│   │   ├── case.escalated                          Timeout escalation
│   │   └── case.coordination.request               Multi-agency
│   │
│   ├── Flow:
│   │   1. Adapter detects trigger (admission, police call)
│   │   2. Event sent to coordination service (< 100ms)
│   │   3. Service enriches with Soc.karta + health context
│   │   4. Notifications sent to all relevant parties
│   │   5. ACK required within deadline (5 min for critical)
│   │   6. Timeout → automatic escalation
│   │
│   └── Audit:
│       ├── Every event logged
│       ├── Every ACK logged
│       └── Every escalation logged
│
├── Notification Service
│   ├── internal/notification/
│   │   ├── service.go                              Multi-channel
│   │   ├── email.go                                Email sender
│   │   ├── sms.go                                  SMS gateway
│   │   └── push.go                                 WebSocket push
│   │
│   └── Channels:
│       ├── WebSocket (real-time dashboard)
│       ├── Email (async notification)
│       └── SMS (critical alerts)
│
└── Context Enrichment
    ├── internal/coordination/enrichment.go
    │
    └── For each case participant:
        ├── Health context (recent hospitalizations, prescriptions)
        ├── Social context (beneficiary status, family, risk)
        └── Case context (open cases in other agencies)
```

**Verifikacija:** Kada pacijent bude primljen u OB Kikinda, CSR automatski dobija notifikaciju sa kontekstom u roku od 30 sekundi.

---

### Phase 7: Pilot Deployment (Kikinda)

**Trajanje: ~4 nedelje**

```
DELIVERABLES:
├── Central Platform (Kragujevac DC)
│   ├── Kubernetes deployment
│   ├── All core services
│   ├── Coordination service
│   └── Monitoring stack
│
├── Edge Deployments
│   ├── OB Kikinda
│   │   ├── Heliant adapter
│   │   ├── Local cache
│   │   └── VPN to Kragujevac
│   │
│   ├── CSR Kikinda
│   │   ├── SOZIS adapter
│   │   ├── Local cache
│   │   └── VPN to Kragujevac
│   │
│   └── DZ Kikinda (optional phase 1)
│       ├── Primary care adapter
│       └── VPN to Kragujevac
│
├── Test Scenarios
│   │
│   ├── Scenario 1: Routine Case Sharing
│   │   1. CSR worker creates case for vulnerable person
│   │   2. Shares with DZ for health assessment
│   │   3. DZ worker views case, adds notes
│   │   4. All actions in audit trail
│   │
│   ├── Scenario 2: Hospital Admission Notification
│   │   1. Patient admitted to OB Kikinda
│   │   2. Adapter detects admission
│   │   3. Platform checks: is this person in CSR system?
│   │   4. If yes → notify CSR worker
│   │   5. CSR worker acknowledges
│   │
│   ├── Scenario 3: Emergency Coordination
│   │   1. Police calls CSR about domestic violence
│   │   2. CSR creates emergency case
│   │   3. Platform enriches with health history
│   │   4. Notifications to all relevant parties
│   │   5. ACK tracking and escalation
│   │
│   └── Scenario 4: Cross-Agency Investigation
│       1. CSR needs health history for court case
│       2. Requests via platform (not phone/paper)
│       3. Health adapter fetches from Heliant
│       4. Data delivered with audit trail
│       5. Court can verify chain of custody
│
└── Documentation
    ├── User guide for CSR workers
    ├── User guide for health workers
    ├── Admin guide
    └── API documentation
```

---

## Updated Architecture

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                            MVP ARCHITECTURE v2.0                                  │
├──────────────────────────────────────────────────────────────────────────────────┤
│                                                                                   │
│   EDGE LOCATIONS (Kikinda)                    CENTRAL (Kragujevac DC)            │
│                                                                                   │
│   ┌─────────────────┐                        ┌────────────────────────────────┐  │
│   │  OB Kikinda     │                        │         API GATEWAY            │  │
│   │  ┌───────────┐  │                        │  (Traefik + Auth + Rate Limit) │  │
│   │  │  Heliant  │  │                        └───────────────┬────────────────┘  │
│   │  │    HIS    │  │                                        │                   │
│   │  └─────┬─────┘  │                        ┌───────────────┴────────────────┐  │
│   │        │        │                        │                                │  │
│   │  ┌─────▼─────┐  │     WebSocket/         │        PLATFORM CORE           │  │
│   │  │  Health   │  │      mTLS              │                                │  │
│   │  │  Adapter  │──┼──────────────────────► │  ┌────────┐  ┌────────┐       │  │
│   │  └───────────┘  │                        │  │ Agency │  │  Case  │       │  │
│   │  [Local Cache]  │                        │  │ Module │  │ Module │       │  │
│   └─────────────────┘                        │  └────────┘  └────────┘       │  │
│                                              │                                │  │
│   ┌─────────────────┐                        │  ┌────────┐  ┌────────┐       │  │
│   │  CSR Kikinda    │                        │  │Document│  │ Audit  │       │  │
│   │  ┌───────────┐  │                        │  │ Module │  │ Module │       │  │
│   │  │   SOZIS   │  │                        │  └────────┘  └────────┘       │  │
│   │  │  Client   │  │                        │                                │  │
│   │  └─────┬─────┘  │                        │  ┌─────────────────────────┐   │  │
│   │        │        │     WebSocket/         │  │  COORDINATION SERVICE   │   │  │
│   │  ┌─────▼─────┐  │      mTLS              │  │                         │   │  │
│   │  │  Social   │──┼──────────────────────► │  │  - Event routing        │   │  │
│   │  │  Adapter  │  │                        │  │  - Context enrichment   │   │  │
│   │  └───────────┘  │                        │  │  - ACK tracking         │   │  │
│   │  [Local Cache]  │                        │  │  - Escalation           │   │  │
│   └─────────────────┘                        │  └─────────────────────────┘   │  │
│                                              │                                │  │
│   ┌─────────────────┐                        │  ┌─────────────────────────┐   │  │
│   │  DZ Kikinda     │                        │  │   NOTIFICATION SERVICE  │   │  │
│   │  ┌───────────┐  │                        │  │                         │   │  │
│   │  │  Primary  │  │     WebSocket/         │  │  - WebSocket push       │   │  │
│   │  │  Care IS  │  │      mTLS              │  │  - Email                │   │  │
│   │  └─────┬─────┘  │                        │  │  - SMS (critical)       │   │  │
│   │        │        │                        │  └─────────────────────────┘   │  │
│   │  ┌─────▼─────┐  │                        │                                │  │
│   │  │ Primary   │──┼──────────────────────► └────────────────────────────────┘  │
│   │  │  Adapter  │  │                                        │                   │
│   │  └───────────┘  │                        ┌───────────────┴────────────────┐  │
│   └─────────────────┘                        │        INFRASTRUCTURE          │  │
│                                              │                                │  │
│   ┌─────────────────┐                        │  ┌────────┐  ┌────────┐       │  │
│   │ Socijalna Karta │                        │  │  NATS  │  │Postgres│       │  │
│   │  (via Servisna  │◄───── Query/Response ──│  │JetStrm │  │   DB   │       │  │
│   │   magistrala)   │                        │  └────────┘  └────────┘       │  │
│   └─────────────────┘                        │                                │  │
│                                              │  ┌────────┐  ┌────────┐       │  │
│                                              │  │  OPA   │  │ MinIO  │       │  │
│                                              │  │ Policy │  │Storage │       │  │
│                                              │  └────────┘  └────────┘       │  │
│                                              │                                │  │
│                                              │  ┌────────┐  ┌────────┐       │  │
│                                              │  │Prometh.│  │Grafana │       │  │
│                                              │  │Metrics │  │ Dash   │       │  │
│                                              │  └────────┘  └────────┘       │  │
│                                              └────────────────────────────────┘  │
│                                                                                   │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Milestones (Updated)

| Milestone | Deliverable | Criteria | Status |
|-----------|-------------|----------|--------|
| **M1** | Foundation Complete | API responds, auth works | ✅ |
| **M2** | Core Modules | Worker creates case with documents | ✅ |
| **M3** | Federation | Cross-agency sharing works | ✅ |
| **M4** | Authorization | OPA policies, metrics | ✅ |
| **M5** | Health Adapter | Heliant data flows to platform | 🔄 |
| **M6** | Social Adapter | SOZIS/Soc.karta integration | 🔄 |
| **M7** | Real-time Coord | Emergency protocol works | 🔄 |
| **M8** | Pilot Ready | Kikinda deployment complete | Pending |

---

## Technical Requirements

### Infrastructure Requirements

| Component | Specification | Location |
|-----------|---------------|----------|
| Kubernetes | 1.28+ (3-node cluster minimum) | Kragujevac DC |
| PostgreSQL | 15+ (8GB RAM, 100GB SSD) | Kragujevac DC |
| NATS | 2.10+ (3-node cluster) | Kragujevac DC |
| MinIO | Latest (50GB initial storage) | Kragujevac DC |
| OPA | Latest | Kragujevac DC |
| Edge Adapters | Docker containers | Kikinda (each institution) |
| VPN | WireGuard/IPsec | Kikinda ↔ Kragujevac |

### Edge Adapter Requirements

| Component | Specification |
|-----------|---------------|
| Runtime | Docker 24+ or Podman |
| OS | Linux (preferred) or Windows Server |
| RAM | 2GB minimum |
| Storage | 10GB for local cache |
| Network | Stable connection to Kragujevac |
| Database Access | Read-only to legacy system |

### Development Requirements

| Aspect | Technology |
|--------|------------|
| Backend | Go 1.22+, Chi router |
| Frontend | React 18+, TypeScript 5+ |
| API Spec | OpenAPI 3.1 |
| Events | AsyncAPI 3.0 |
| Health Data | FHIR R4 |
| CI/CD | GitLab CE |

---

## Risk Assessment (Updated)

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Heliant DB access denied | High | Medium | Early engagement with vendor, fallback to file export |
| SOZIS API unavailable | High | Medium | Direct DB read if approved, manual sync fallback |
| Network instability Kikinda | Medium | Medium | Local cache, retry logic, offline mode |
| CSR staff resistance | Medium | Low | Training, gradual rollout, show time savings |
| Data privacy concerns | High | Medium | OPA policies, audit trail, data minimization |
| Vendor lock-in (Asseco) | Medium | Low | Standard protocols (FHIR), adapter abstraction |

---

## Success Criteria for MVP (Updated)

### Technical Criteria

- [ ] Platform API responds with < 500ms (p95)
- [ ] Health adapter successfully reads from Heliant
- [ ] Social adapter successfully reads from SOZIS
- [ ] Socijalna karta API integration works
- [ ] Real-time events delivered in < 100ms
- [ ] Emergency ACK within 5 minutes tracked
- [ ] All actions in immutable audit log
- [ ] Zero data breaches during pilot

### Operational Criteria

- [ ] Hospital admission triggers CSR notification
- [ ] CSR worker can view patient health context
- [ ] Cross-agency case sharing works end-to-end
- [ ] Emergency escalation protocol functions
- [ ] Audit trail verifiable by admin

### Business Criteria

- [ ] CSR workers report time savings
- [ ] Health workers report easier coordination
- [ ] No increase in phone calls for coordination
- [ ] Documented reduction in response time for emergencies
- [ ] Positive feedback from pilot participants

---

## Estimated Timeline (Updated)

| Phase | Duration | Status |
|-------|----------|--------|
| Phase 1: Foundation | 4 weeks | ✅ Complete |
| Phase 2: Core Modules | 6 weeks | ✅ Complete |
| Phase 3: Federation | 4 weeks | ✅ Complete |
| Phase 4: Authorization | 3 weeks | ✅ Complete |
| Phase 5: Legacy Adapters | 6 weeks | 🔄 Next |
| Phase 6: Real-time Coordination | 4 weeks | Pending |
| Phase 7: Pilot Deployment | 4 weeks | Pending |
| **Total** | **~31 weeks** | |
| **Remaining** | **~14 weeks** | |

---

## Resource Requirements (Updated)

### Team Composition

| Role | Count | Responsibility |
|------|-------|----------------|
| Tech Lead | 1 | Architecture, code review |
| Backend Developer | 2 | Go modules, adapters |
| Integration Specialist | 1 | Legacy system integration |
| Frontend Developer | 1 | React UI |
| DevOps Engineer | 1 | K8s, CI/CD, edge deployment |
| Security Specialist | 0.5 | Reviews, OPA policies |
| QA Engineer | 1 | Testing, documentation |
| **Pilot Coordinator** | 1 | Stakeholder management, training |

### External Dependencies

| Dependency | Owner | Status |
|------------|-------|--------|
| Heliant DB access | OB Kikinda IT | To be requested |
| SOZIS API/DB access | MINRZS/Asseco | To be requested |
| Socijalna karta API | Servisna magistrala | To be requested |
| VPN setup | OITeG | To be requested |
| Kragujevac DC resources | OITeG | To be requested |

---

## Post-MVP Roadmap

Nakon uspešnog MVP-a u Kikindi:

### Phase 6: Extended Integration
1. **GIZ Adapter** - Rezidencijalne ustanove (Dom za stare Kikinda)
2. **Full eID Integration** - Registracija kod eid.gov.rs
3. **Qualified Signatures** - eIDAS QES

### Phase 7: Scale to Region
1. **Additional hospitals** - KC Vojvodina, OB Zrenjanin
2. **Additional CSRs** - Zrenjanin, Novi Sad
3. **Police integration** - MUP adapter (ako 112 ne bude operativan)

### Phase 8: National Rollout
1. **All KC hospitals** - National coverage
2. **All CSRs** - 171 centers
3. **Gerontology centers** - Integration with residential care
4. **Mobile applications** - React Native apps
5. **Workflow Engine** - Temporal for complex processes

---

## Immediate Next Steps

### Week 1-2: Adapter Framework
1. Create `internal/adapters/` package structure
2. Define Health adapter interface
3. Define Social adapter interface
4. Implement FHIR transformation utilities

### Week 3-4: Heliant Adapter
1. Get Heliant DB schema documentation
2. Implement read-only queries
3. Implement real-time polling
4. Test with sample data

### Week 5-6: Social Adapters
1. Get Socijalna karta API documentation
2. Implement mTLS client
3. Implement SOZIS adapter (if access granted)
4. Test integration

### Week 7-8: Coordination Service
1. Implement event routing
2. Implement context enrichment
3. Implement ACK tracking
4. Implement escalation

### Week 9-10: Notification Service
1. Implement WebSocket push
2. Implement email notifications
3. Implement SMS for critical alerts
4. Test end-to-end flow

### Week 11-14: Pilot Deployment
1. Deploy to Kragujevac DC
2. Deploy edge adapters to Kikinda
3. Training for pilot users
4. Run test scenarios
5. Collect feedback
6. Iterate

---

## Conclusion

MVP platforma demonstriraće:

1. **Tehničku izvodljivost** - Moderna arhitektura sa edge adapterima
2. **Real-time koordinaciju** - Ono što eZUP ne može
3. **Health-social integraciju** - Povezivanje IZIS i SOZIS
4. **Praktičnu vrednost** - Merljivo smanjenje vremena koordinacije
5. **Bezbednost** - Kompletna revizijska traga, OPA policies
6. **Skalabilnost** - Temelj za nacionalno proširenje

**Ključna razlika od postojećih sistema**: Ova platforma nije zamena za SOZIS ili IZIS - ona ih **povezuje** kroz standardizovane adaptere i omogućava **real-time koordinaciju** koju trenutno nijedan sistem ne pruža.

---

## Appendices

### A. Related Documents

| Document | Description |
|----------|-------------|
| `docs/adapter-architecture.md` | Detailed adapter architecture |
| `docs/tech-stack.md` | Technology decisions |
| `docs/domain-model.md` | Domain entities |
| `docs/event-catalog.md` | Event types |
| `docs/security-model.md` | Security policies |
| `docs/it-sistem-srb-research-init.md` | Research on Serbian IT fragmentation |
| `docs/outreach-letters.md` | Outreach letters to institutions |

### B. Pilot Location: Why Kikinda?

1. **Dokumentovani problemi** - Slučaj iz pisama
2. **Upravljiva veličina** - ~40,000 stanovnika
3. **Pozitivan primer** - Osnovni sud pokazao spremnost na saradnju
4. **Kompletna infrastruktura** - Bolnica, CSR, DZ u istom gradu
5. **Lokalno znanje** - Autor projekta iz Kikinde

### C. FHIR Resources for Serbia

| Serbian Concept | FHIR Resource | Serbian Profile |
|-----------------|---------------|-----------------|
| Pacijent | Patient | SerbianPatient (JMBG, LBO) |
| Hospitalizacija | Encounter | SerbianEncounter |
| Dijagnoza | Condition | ICD-10 coded |
| Recept | MedicationRequest | ATC coded |
| Laboratorija | Observation | LOINC coded |
| Upućivanje | ServiceRequest | SerbianReferral |

---

*Dokument pripremljen za prezentaciju Vladi Republike Srbije*
*Verzija 2.0 - Ažurirano sa adapter arhitekturom i Kikinda pilotom*
