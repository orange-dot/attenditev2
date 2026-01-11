# Security Model

> Serbia Government Interoperability Platform

## Overview

This document defines the authentication, authorization, roles, permissions, and data access policies for the platform.

---

## Authentication

### Identity Sources

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      AUTHENTICATION FLOW                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐                                                       │
│  │   Citizens   │───► Serbia eID ───┐                                   │
│  └──────────────┘                    │                                   │
│                                      ▼                                   │
│  ┌──────────────┐              ┌──────────┐     ┌──────────────┐        │
│  │   Workers    │───► Agency ──│ Keycloak │────►│   Platform   │        │
│  └──────────────┘    LDAP/AD   │  (IdP)   │     │   (JWT)      │        │
│                                      ▲          └──────────────┘        │
│  ┌──────────────┐                    │                                   │
│  │   Admins     │───► Local ─────────┘                                   │
│  └──────────────┘    Accounts                                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Authentication Methods

| User Type | Method | Assurance Level |
|-----------|--------|-----------------|
| Citizen | Serbia eID (ConsentID) | High |
| Citizen | Serbia eID (QES) | Highest |
| Worker | Agency LDAP/AD + MFA | High |
| Admin | Local account + MFA | High |
| System | mTLS certificate | Highest |

### Session Management

| Parameter | Value |
|-----------|-------|
| Access Token TTL | 15 minutes |
| Refresh Token TTL | 8 hours |
| Idle Timeout | 30 minutes |
| Absolute Timeout | 12 hours |
| Concurrent Sessions | 3 per user |

### Multi-Factor Authentication

**Required for:**
- All workers (agency employees)
- All admins
- Citizens accessing sensitive services

**MFA Methods:**
- TOTP (authenticator app)
- ConsentID push notification
- SMS (fallback only)

---

## Authorization Model

### Hybrid RBAC + ABAC

```
┌─────────────────────────────────────────────────────────────────────────┐
│                      AUTHORIZATION FLOW                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Request ──► API Gateway ──► OPA Policy Engine ──► Allow/Deny           │
│                                     │                                    │
│                    ┌────────────────┼────────────────┐                  │
│                    │                │                │                  │
│                    ▼                ▼                ▼                  │
│               ┌────────┐      ┌──────────┐    ┌───────────┐            │
│               │  RBAC  │      │   ABAC   │    │  Context  │            │
│               │ (Role) │      │(Attribute)│   │ (Dynamic) │            │
│               └────────┘      └──────────┘    └───────────┘            │
│                                                                          │
│  Factors:                                                               │
│  • User role (worker, admin, citizen)                                   │
│  • User agency                                                          │
│  • Resource ownership                                                   │
│  • Resource sharing permissions                                         │
│  • Time of day                                                          │
│  • IP/location                                                          │
│  • Case sensitivity level                                               │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Roles

### System Roles

| Role | Description | Scope |
|------|-------------|-------|
| `platform_admin` | Full platform access | Global |
| `platform_operator` | Operations, monitoring | Global |
| `security_auditor` | Read-only audit access | Global |

### Agency Roles

| Role | Description | Scope |
|------|-------------|-------|
| `agency_admin` | Manage agency workers, settings | Agency |
| `agency_supervisor` | Supervise workers, escalations | Agency |
| `case_worker` | Handle cases | Agency |
| `dispatch_operator` | Dispatch console access | Agency |
| `field_unit` | Mobile field worker | Agency |
| `agency_viewer` | Read-only agency access | Agency |

### Case Roles (Per-case assignment)

| Role | Description |
|------|-------------|
| `lead` | Primary responsible |
| `support` | Assisting |
| `reviewer` | Approval authority |
| `observer` | Read-only |

### Citizen Roles

| Role | Description |
|------|-------------|
| `citizen` | Basic authenticated citizen |
| `citizen_verified` | eID-verified citizen |

---

## Permissions

### Permission Naming

```
{resource}.{action}

Examples:
- case.create
- case.read
- case.update
- case.delete
- case.transfer
- case.assign
```

### Permission Matrix

#### Case Permissions

| Permission | platform_admin | agency_admin | agency_supervisor | case_worker | citizen |
|------------|----------------|--------------|-------------------|-------------|---------|
| case.create | ✓ | ✓ | ✓ | ✓ | ✓* |
| case.read | ✓ | ✓ | ✓ | ✓** | ✓*** |
| case.update | ✓ | ✓ | ✓ | ✓** | |
| case.delete | ✓ | | | | |
| case.assign | ✓ | ✓ | ✓ | | |
| case.transfer | ✓ | ✓ | ✓ | | |
| case.close | ✓ | ✓ | ✓ | ✓** | |
| case.escalate | ✓ | ✓ | ✓ | ✓** | |

```
* Citizens can create cases (requests/applications)
** Only if assigned to case
*** Only own cases
```

#### Dispatch Permissions

| Permission | platform_admin | agency_admin | dispatch_operator | field_unit |
|------------|----------------|--------------|-------------------|------------|
| incident.create | ✓ | ✓ | ✓ | ✓ |
| incident.read | ✓ | ✓ | ✓ | ✓* |
| incident.update | ✓ | ✓ | ✓ | |
| incident.close | ✓ | ✓ | ✓ | |
| unit.dispatch | ✓ | ✓ | ✓ | |
| unit.status.update | ✓ | ✓ | ✓ | ✓** |
| unit.location.update | | | | ✓** |

```
* Only assigned incidents
** Only own unit
```

#### Document Permissions

| Permission | platform_admin | agency_admin | case_worker | citizen |
|------------|----------------|--------------|-------------|---------|
| document.create | ✓ | ✓ | ✓ | ✓* |
| document.read | ✓ | ✓ | ✓** | ✓*** |
| document.update | ✓ | ✓ | ✓** | |
| document.delete | ✓ | ✓ | | |
| document.sign | ✓ | ✓ | ✓** | ✓**** |

```
* Upload documents to own cases
** Documents in assigned cases
*** Own documents
**** If signature requested
```

#### Admin Permissions

| Permission | platform_admin | agency_admin | security_auditor |
|------------|----------------|--------------|------------------|
| agency.create | ✓ | | |
| agency.update | ✓ | ✓* | |
| agency.delete | ✓ | | |
| worker.create | ✓ | ✓* | |
| worker.update | ✓ | ✓* | |
| worker.delete | ✓ | ✓* | |
| audit.read | ✓ | | ✓ |
| audit.export | ✓ | | ✓ |

```
* Own agency only
```

---

## OPA Policies

### Policy Structure

```
/policies
├── /common
│   ├── roles.rego        # Role definitions
│   └── helpers.rego      # Utility functions
├── /case
│   ├── access.rego       # Case access rules
│   └── actions.rego      # Case action rules
├── /dispatch
│   ├── access.rego
│   └── actions.rego
├── /document
│   ├── access.rego
│   └── actions.rego
└── /admin
    └── access.rego
```

### Example: Case Access Policy

```rego
# /policies/case/access.rego
package case.access

import future.keywords.in

default allow = false

# Platform admins can access all cases
allow {
    input.user.roles[_] == "platform_admin"
}

# Agency admins can access cases owned by their agency
allow {
    input.user.roles[_] == "agency_admin"
    input.resource.owning_agency == input.user.agency_id
}

# Workers can access cases they are assigned to
allow {
    input.user.roles[_] == "case_worker"
    input.user.id == input.resource.assignments[_].worker_id
}

# Workers can access cases shared with their agency
allow {
    input.user.roles[_] == "case_worker"
    input.user.agency_id in input.resource.shared_with
    agency_access_level(input.user.agency_id, input.resource) >= required_level(input.action)
}

# Citizens can access their own cases (as participant)
allow {
    input.user.roles[_] == "citizen"
    input.user.citizen_id == input.resource.participants[_].citizen_id
}

# Helper: Get agency access level for a case
agency_access_level(agency_id, case_resource) = level {
    level := case_resource.access_level[agency_id]
}

# Helper: Map action to required access level
required_level(action) = 1 { action == "read" }
required_level(action) = 2 { action == "comment" }
required_level(action) = 3 { action in ["update", "assign"] }
required_level(action) = 4 { action in ["transfer", "close"] }
```

### Example: Dispatch Policy

```rego
# /policies/dispatch/actions.rego
package dispatch.actions

default allow_dispatch = false

# Only dispatch operators can dispatch units
allow_dispatch {
    input.user.roles[_] == "dispatch_operator"
    input.user.agency_id == input.unit.agency_id
    input.unit.status == "available"
}

# Can dispatch to incidents in jurisdiction
allow_dispatch {
    input.user.roles[_] == "dispatch_operator"
    incident_in_jurisdiction(input.incident, input.user.agency_id)
}

# Multi-agency dispatch requires coordinator role
allow_dispatch {
    input.user.roles[_] == "dispatch_coordinator"
    input.unit.agency_id != input.user.agency_id
}
```

---

## Data Access Rules

### Cross-Agency Data Sharing

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    CROSS-AGENCY ACCESS MODEL                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Case owned by Agency A, shared with Agency B:                          │
│                                                                          │
│  ┌──────────────┐         ┌──────────────┐                              │
│  │   Agency A   │         │   Agency B   │                              │
│  │   (Owner)    │         │  (Shared)    │                              │
│  │              │         │              │                              │
│  │  Full Access │────────►│ Access Level │                              │
│  │              │ Shares  │              │                              │
│  └──────────────┘         └──────────────┘                              │
│                                                                          │
│  Access Levels:                                                         │
│  ├── 0: None (revoked)                                                  │
│  ├── 1: Read (view case, documents)                                     │
│  ├── 2: Comment (add notes, messages)                                   │
│  ├── 3: Contribute (update, assign own workers)                         │
│  └── 4: Full (all except transfer ownership)                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Data Classification

| Level | Label | Example | Access |
|-------|-------|---------|--------|
| 0 | Public | Agency contact info | Anyone |
| 1 | Internal | Case statistics | Authenticated workers |
| 2 | Confidential | Case details | Assigned workers |
| 3 | Restricted | Medical records | Need-to-know + audit |
| 4 | Secret | Witness protection | Special clearance |

### Field-Level Access Control

Some fields have additional restrictions:

```go
type Case struct {
    ID          string            // Level 1
    CaseNumber  string            // Level 1
    Type        string            // Level 1
    Status      string            // Level 1
    Title       string            // Level 2
    Description string            // Level 2

    // Restricted fields
    Participants []Participant    // Level 2 (names), Level 3 (details)

    // Special handling
    JMBG        string            // Level 3, masked unless authorized
}
```

### JMBG (Personal ID) Handling

```go
// JMBG access rules
func CanAccessFullJMBG(user User, purpose string) bool {
    // Only specific roles can see full JMBG
    if !hasRole(user, "case_worker", "agency_admin", "platform_admin") {
        return false
    }

    // Must provide business justification
    if purpose == "" {
        return false
    }

    // Log access
    auditLog.Log(AuditEntry{
        Action:       "jmbg.access",
        ActorID:      user.ID,
        Justification: purpose,
    })

    return true
}

// Masked JMBG for display: "0101990******"
func MaskJMBG(jmbg string) string {
    if len(jmbg) < 13 {
        return "***********"
    }
    return jmbg[:7] + "******"
}
```

---

## API Security

### Request Authentication

```
Authorization: Bearer <JWT>
X-Request-ID: <UUID>
X-Correlation-ID: <UUID>
```

### JWT Claims

```json
{
  "sub": "user-uuid",
  "iss": "keycloak",
  "aud": "gov-platform",
  "exp": 1234567890,
  "iat": 1234567890,

  "user_type": "worker",
  "agency_id": "agency-uuid",
  "roles": ["case_worker", "dispatch_operator"],
  "permissions": ["case.read", "case.update", "incident.create"],

  "eid_verified": true,
  "eid_assurance": "high",

  "session_id": "session-uuid",
  "mfa_verified": true
}
```

### Rate Limiting

| Endpoint Type | Limit | Window |
|---------------|-------|--------|
| Authentication | 10 | 1 minute |
| API (authenticated) | 1000 | 1 minute |
| Search | 100 | 1 minute |
| File upload | 20 | 1 minute |
| Bulk operations | 10 | 1 minute |

### Input Validation

```go
// All inputs validated and sanitized
type CreateCaseRequest struct {
    Type        CaseType `validate:"required,oneof=CHILD_WELFARE CRIMINAL ..."`
    Priority    Priority `validate:"required,oneof=LOW MEDIUM HIGH URGENT"`
    Title       string   `validate:"required,min=10,max=200,no_html"`
    Description string   `validate:"required,min=50,max=10000,no_html"`
}
```

---

## Security Controls

### Transport Security

| Control | Implementation |
|---------|----------------|
| TLS Version | 1.3 only |
| Cipher Suites | TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256 |
| Certificate Pinning | For mobile apps |
| HSTS | Enabled, 1 year, includeSubDomains |

### Application Security

| Control | Implementation |
|---------|----------------|
| CSRF | Double-submit cookie |
| XSS | CSP headers, output encoding |
| SQL Injection | Parameterized queries only |
| Path Traversal | Whitelist validation |
| File Upload | Type validation, size limits, virus scan |

### Secrets Management

| Secret Type | Storage | Rotation |
|-------------|---------|----------|
| API Keys | OpenBao | 90 days |
| DB Credentials | OpenBao | 30 days (dynamic) |
| JWT Signing Key | OpenBao | 7 days |
| Encryption Keys | OpenBao (HSM) | 1 year |
| TLS Certificates | cert-manager | 90 days (auto) |

---

## Audit Requirements

### What Must Be Logged

| Category | Events |
|----------|--------|
| Authentication | Login, logout, failed attempts, MFA |
| Authorization | Access denied, privilege escalation |
| Data Access | Read sensitive data, export |
| Data Modification | Create, update, delete |
| Admin Actions | User management, config changes |
| Security Events | Suspicious activity, rate limit hits |

### Audit Log Fields

```go
type AuditEntry struct {
    // Identity
    Timestamp     time.Time
    ActorType     string    // citizen, worker, system
    ActorID       string
    ActorAgency   string
    SessionID     string

    // Request
    RequestID     string
    CorrelationID string
    IPAddress     string
    UserAgent     string

    // Action
    Action        string    // e.g., "case.read"
    ResourceType  string    // e.g., "case"
    ResourceID    string

    // Context
    Justification string    // Business reason (for sensitive access)
    Result        string    // success, denied, error

    // Changes (for modifications)
    OldValue      any
    NewValue      any
}
```

### Retention

| Data Type | Retention |
|-----------|-----------|
| Authentication logs | 2 years |
| Authorization logs | 7 years |
| Case access logs | 10 years |
| Criminal case logs | 30 years |
| Audit exports | Permanent |

---

## Data Privacy Architecture (Pseudonymization)

### Design Principle: "Sistem zna samo što mora"

Centralni sistem radi sa **pseudonimizovanim podacima** - ima pristup samo internom ID-u bez mogućnosti identifikacije osobe. Lični podaci ostaju isključivo pod kontrolom lokalne ustanove.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    DATA PRIVACY ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    LOKALNA USTANOVA (Data Controller)              │ │
│  │                                                                    │ │
│  │  ┌─────────────────┐    ┌────────────────────────────────────────┐│ │
│  │  │  Puni podaci    │    │  Pseudonymization Service              ││ │
│  │  │                 │    │                                        ││ │
│  │  │  • JMBG         │    │  JMBG ──────► PseudonymID (UUID)      ││ │
│  │  │  • Ime/Prezime  │    │  Ime ───────► [ne šalje se]           ││ │
│  │  │  • Adresa       │    │  Adresa ────► [ne šalje se]           ││ │
│  │  │  • Kontakt      │    │  Medicinski ► hashirani/agregirani    ││ │
│  │  │  • Medicinski   │    │                                        ││ │
│  │  └─────────────────┘    └────────────────────────────────────────┘│ │
│  │                                      │                             │ │
│  │            Ključ za de-pseudonimizaciju ostaje OVDE               │ │
│  └──────────────────────────────────────┼─────────────────────────────┘ │
│                                         │                               │
│                                         ▼                               │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                    CENTRALNI SISTEM (Data Processor)               │ │
│  │                                                                    │ │
│  │  Ima pristup SAMO:                    NEMA pristup:               │ │
│  │  ┌─────────────────┐                  ┌─────────────────┐         │ │
│  │  │ • PseudonymID   │                  │ ✗ JMBG          │         │ │
│  │  │ • Tip slučaja   │                  │ ✗ Ime/Prezime   │         │ │
│  │  │ • Status        │                  │ ✗ Adresa        │         │ │
│  │  │ • Vremenske     │                  │ ✗ Kontakt       │         │ │
│  │  │   oznake        │                  │ ✗ Bilo šta što  │         │ │
│  │  │ • Agregirani    │                  │   identifikuje  │         │ │
│  │  │   medicinski    │                  │   osobu         │         │ │
│  │  └─────────────────┘                  └─────────────────┘         │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Entiteti za centralni sistem

```go
// Centralni sistem NIKADA ne vidi ovo:
type Citizen struct {
    JMBG        string  // ✗ NE ŠALJE SE
    FirstName   string  // ✗ NE ŠALJE SE
    LastName    string  // ✗ NE ŠALJE SE
    Address     Address // ✗ NE ŠALJE SE
    Contact     Contact // ✗ NE ŠALJE SE
}

// Centralni sistem vidi SAMO ovo:
type PseudonymizedSubject struct {
    PseudonymID     string    // UUID bez značenja, ne može se dekodirati
    LocalFacilityID string    // Koja ustanova "poseduje" subjekta
    CreatedAt       time.Time

    // Opciono: samo ako je neophodno za funkcionalnost
    AgeGroup        string    // "18-25", "26-35", itd. (ne tačan datum)
    Region          string    // Oblast, ne tačna adresa
}
```

### De-pseudonimizacija protokol

Kada je identifikacija neophodna (npr. hitni slučaj), zahtev mora proći kroz strogu proceduru:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    DE-PSEUDONYMIZATION FLOW                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Zahtev                                                              │
│     ┌──────────────┐                                                    │
│     │ Ovlašćeno    │──► Zahtev mora sadržati:                          │
│     │ lice/sistem  │    • PseudonymID                                   │
│     └──────────────┘    • Razlog (purpose)                              │
│            │            • Pravni osnov                                  │
│            │            • Vremensko ograničenje                         │
│            ▼                                                            │
│  2. Verifikacija                                                        │
│     ┌──────────────┐                                                    │
│     │ Lokalna      │──► Provera:                                        │
│     │ ustanova     │    • Da li je zahtev legitiman?                   │
│     └──────────────┘    • Da li lice ima ovlašćenje?                   │
│            │            • Da li postoji pravni osnov?                   │
│            │                                                            │
│            ▼                                                            │
│  3. Audit log (OBAVEZAN)                                                │
│     ┌──────────────┐                                                    │
│     │ audit.depseud│    Trajno se beleži:                              │
│     │ onymization  │    • Ko je tražio                                 │
│     └──────────────┘    • Kada                                          │
│            │            • Za koga (PseudonymID)                         │
│            │            • Zašto                                         │
│            ▼            • Rezultat (odobren/odbijen)                    │
│  4. Odgovor                                                             │
│     ┌──────────────┐                                                    │
│     │ Siguran      │──► Podaci se vraćaju samo kroz:                   │
│     │ kanal        │    • Enkriptovani kanal                           │
│     └──────────────┘    • Sa vremenskim ograničenjem                   │
│                         • Bez keširanja                                 │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### AI sistem - minimalni pristup

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    AI SYSTEM DATA ACCESS                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  AI sistem po defaultu:                                                 │
│  ━━━━━━━━━━━━━━━━━━━━━                                                  │
│  ✓ Agregirani statistički podaci                                       │
│  ✓ Anonimizovani uzorci za trening                                     │
│  ✓ Pseudonimizovani podaci za predikcije                               │
│  ✗ NIKADA direktan pristup ličnim podacima                             │
│                                                                          │
│  Nivo pristupa:                                                         │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  LEVEL 0: Agregirano         Broj slučajeva po regionu/tipu       │ │
│  │           (default)           Prosečna vremena obrade             │ │
│  │                               Trendovi bez identifikacije         │ │
│  ├────────────────────────────────────────────────────────────────────┤ │
│  │  LEVEL 1: Pseudonimizovano   PseudonymID + atributi slučaja       │ │
│  │           (sa odobrenjem)     Za personalizovane preporuke        │ │
│  │                               Bez mogućnosti identifikacije       │ │
│  ├────────────────────────────────────────────────────────────────────┤ │
│  │  LEVEL 2: Linkabilno         Samo u hitnim slučajevima           │ │
│  │           (izuzetak)          Zahteva: sudski nalog ili           │ │
│  │                               životna opasnost + audit            │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Implementacija pseudonimizacije

```go
// /internal/privacy/pseudonymization.go

// PseudonymizationService - ostaje u lokalnoj ustanovi
type PseudonymizationService struct {
    // Ključ za HMAC - NIKADA ne napušta lokalnu ustanovu
    secretKey []byte

    // Lokalna baza mapiranja
    mappingStore MappingStore
}

// Generisanje pseudonima - jednosmerno bez ključa
func (s *PseudonymizationService) CreatePseudonym(jmbg string) string {
    // HMAC-SHA256 sa tajnim ključem
    // Isti JMBG uvek daje isti pseudonim (za konzistentnost)
    // Ali bez ključa - nemoguće je rekonstruisati JMBG
    h := hmac.New(sha256.New, s.secretKey)
    h.Write([]byte(jmbg))
    return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

// De-pseudonimizacija - zahteva audit
func (s *PseudonymizationService) ResolvePseudonym(
    ctx context.Context,
    pseudonymID string,
    request DepseudonymizationRequest,
) (*CitizenIdentity, error) {

    // 1. Verifikacija ovlašćenja
    if !s.isAuthorized(ctx, request) {
        audit.Log(AuditEntry{
            Action:   "depseudonymization.denied",
            Reason:   "unauthorized",
            // ...
        })
        return nil, ErrUnauthorized
    }

    // 2. Obavezan audit log
    audit.Log(AuditEntry{
        Action:        "depseudonymization.executed",
        PseudonymID:   pseudonymID,
        RequestedBy:   request.RequestorID,
        Purpose:       request.Purpose,
        LegalBasis:    request.LegalBasis,
        Timestamp:     time.Now(),
    })

    // 3. Vraćanje podataka samo ako je odobreno
    return s.mappingStore.Resolve(pseudonymID)
}

// Zahtev za de-pseudonimizaciju
type DepseudonymizationRequest struct {
    RequestorID   string
    RequestorRole string
    Purpose       string        // Zašto je potrebno
    LegalBasis    string        // Pravni osnov
    CaseID        string        // Povezani slučaj
    Expiry        time.Duration // Koliko dugo važi pristup
    IsEmergency   bool          // Hitni slučaj (brža procedura)
}
```

### Pravila za centralni sistem

| Podatak | Pristup centralnog sistema | Napomena |
|---------|---------------------------|----------|
| PseudonymID | ✓ Uvek | Osnovni identifikator |
| Tip slučaja | ✓ Uvek | Potrebno za rutiranje |
| Status | ✓ Uvek | Praćenje toka |
| Vremena | ✓ Uvek | SLA, statistika |
| Ustanova | ✓ Uvek | Koordinacija |
| JMBG | ✗ Nikada | Ostaje lokalno |
| Ime/Prezime | ✗ Nikada | Ostaje lokalno |
| Adresa | ✗ Nikada | Samo region ako treba |
| Kontakt | ✗ Nikada | Ostaje lokalno |
| Medicinski detalji | ⚠ Agregirano | Samo ako mora, bez identifikacije |

### Bezbednosne kontrole

```go
// Middleware koji sprečava slanje ličnih podataka
func PrivacyGuardMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Intercept response
        rec := httptest.NewRecorder()
        next.ServeHTTP(rec, r)

        body := rec.Body.Bytes()

        // Provera da li odgovor sadrži zabranjene podatke
        if containsPersonalData(body) {
            // Log security incident
            securityLog.Alert("personal_data_leak_attempt", map[string]any{
                "endpoint": r.URL.Path,
                "method":   r.Method,
            })

            http.Error(w, "Privacy violation blocked", http.StatusForbidden)
            return
        }

        // Kopiranje odgovora
        for k, v := range rec.Header() {
            w.Header()[k] = v
        }
        w.WriteHeader(rec.Code)
        w.Write(body)
    })
}

// Detekcija ličnih podataka u odgovoru
func containsPersonalData(data []byte) bool {
    // Regex za JMBG format
    jmbgPattern := regexp.MustCompile(`\b\d{13}\b`)
    if jmbgPattern.Match(data) {
        return true
    }

    // Ostali obrasci: email, telefon u srpskom formatu, itd.
    // ...

    return false
}
```

### Prednosti ovog pristupa

1. **GDPR usklađenost**: Pseudonimizacija je eksplicitno preporučena mera
2. **Minimizacija podataka**: Centralni sistem ima samo što mora
3. **Decentralizacija rizika**: Kompromitovanje centralnog sistema ne otkriva identitete
4. **Audit trail**: Svaki pokušaj identifikacije je zabeležen
5. **Lokalna kontrola**: Ustanova zadržava kontrolu nad svojim podacima

---

## Compliance Mapping

### GDPR

| Requirement | Implementation |
|-------------|----------------|
| Lawful basis | Documented per processing activity |
| Data minimization | Pseudonimizacija - centralni sistem ima samo PseudonymID |
| Purpose limitation | Access logs include purpose; de-pseudonimizacija zahteva razlog |
| Right to access | Citizen portal, audit log |
| Right to erasure | Anonymization workflow; pseudonim se može obrisati bez uticaja na centralni sistem |
| Data breach notification | Incident response procedure |
| Privacy by design | Arhitektura sa pseudonimizacijom od početka |
| Data protection by default | Lični podaci nikada ne napuštaju lokalnu ustanovu bez eksplicitnog zahteva |

### eIDAS

| Requirement | Implementation |
|-------------|----------------|
| Identity assurance | Serbia eID levels mapped |
| Qualified signatures | QES via cloud certificates |
| Timestamp authority | TSA integration |
| Audit trail | Complete transaction logs |

---

## Security Checklist

### Before Production

- [ ] Penetration test completed
- [ ] Security audit completed
- [ ] OPA policies reviewed
- [ ] All secrets in OpenBao
- [ ] TLS properly configured
- [ ] Rate limiting enabled
- [ ] Audit logging verified
- [ ] Backup encryption verified
- [ ] Incident response plan documented
- [ ] Security training completed

### Ongoing

- [ ] Weekly vulnerability scans
- [ ] Monthly access reviews
- [ ] Quarterly penetration tests
- [ ] Annual security audit
- [ ] Continuous monitoring alerts
