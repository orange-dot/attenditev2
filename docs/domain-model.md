# Domain Model

> Serbia Government Interoperability Platform

## Overview

This document defines the core domain entities, their relationships, and business rules.

---

## Bounded Contexts

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         BOUNDED CONTEXTS                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │      CASE       │  │    DISPATCH     │  │    IDENTITY     │         │
│  │   MANAGEMENT    │  │   (Emergency)   │  │   (Reference)   │         │
│  │                 │  │                 │  │                 │         │
│  │  • Case         │  │  • Incident     │  │  • Citizen      │         │
│  │  • CaseEvent    │  │  • Unit         │  │  • Agency       │         │
│  │  • Participant  │  │  • Dispatch     │  │  • Worker       │         │
│  │  • Assignment   │  │  • Location     │  │                 │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│                                                                          │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │    DOCUMENT     │  │   MESSAGING     │  │     AUDIT       │         │
│  │   MANAGEMENT    │  │                 │  │                 │         │
│  │                 │  │                 │  │                 │         │
│  │  • Document     │  │  • Conversation │  │  • AuditEntry   │         │
│  │  • Version      │  │  • Message      │  │                 │         │
│  │  • Signature    │  │  • Participant  │  │                 │         │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘         │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Identity Context (CRUD)

Reference data for people and organizations.

### Citizen

```
┌─────────────────────────────────────────┐
│               CITIZEN                    │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  jmbg            : string (13 digits)    │  ← Serbian personal ID
│  first_name      : string                │
│  last_name       : string                │
│  date_of_birth   : date                  │
│  gender          : enum                  │
│  address         : Address               │
│  contact         : ContactInfo           │
│  eid_linked      : boolean               │  ← Serbia eID linked
│  eid_assurance   : enum                  │  ← basic/high/highest
│  created_at      : timestamp             │
│  updated_at      : timestamp             │
└─────────────────────────────────────────┘

Address {
  street      : string
  city        : string
  postal_code : string
  country     : string (default: "RS")
  coordinates : Point (lat, lng)  ← PostGIS
}

ContactInfo {
  email       : string
  phone       : string
  mobile      : string
}
```

**Business Rules:**
- JMBG must be unique and valid (checksum)
- JMBG format: 13 digits (DDMMYYYRRBBBK)
- Citizen can be linked to Serbia eID after identity verification

---

### Agency

```
┌─────────────────────────────────────────┐
│                AGENCY                    │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  code            : string (unique)       │  ← e.g., "MUP", "RFZO"
│  name            : string                │
│  type            : AgencyType            │
│  parent_id       : UUID (nullable)       │  ← Hierarchy
│  address         : Address               │
│  contact         : ContactInfo           │
│  jurisdiction    : Polygon               │  ← PostGIS
│  status          : enum                  │
│  federation_cert : bytes                 │  ← For mTLS
│  created_at      : timestamp             │
│  updated_at      : timestamp             │
└─────────────────────────────────────────┘

AgencyType: enum {
  POLICE
  HEALTHCARE
  SOCIAL_SERVICES
  JUDICIARY
  TAX
  LOCAL_GOVERNMENT
  EDUCATION
  EMERGENCY
  OTHER
}
```

**Business Rules:**
- Agency code must be unique
- Parent agency must exist if specified
- Federation certificate required for inter-agency communication

---

### Worker

```
┌─────────────────────────────────────────┐
│                WORKER                    │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  agency_id       : UUID (FK)             │
│  citizen_id      : UUID (FK, nullable)   │  ← Link to citizen record
│  employee_id     : string                │  ← Agency internal ID
│  first_name      : string                │
│  last_name       : string                │
│  email           : string                │
│  position        : string                │
│  department      : string                │
│  roles           : []WorkerRole          │
│  status          : enum                  │
│  created_at      : timestamp             │
│  updated_at      : timestamp             │
└─────────────────────────────────────────┘

WorkerRole {
  role        : string     ← e.g., "case_worker", "supervisor"
  scope       : string     ← e.g., "child_welfare", "all"
  granted_at  : timestamp
  granted_by  : UUID
}

WorkerStatus: enum {
  ACTIVE
  ON_LEAVE
  SUSPENDED
  TERMINATED
}
```

**Business Rules:**
- Worker belongs to exactly one agency
- Worker can have multiple roles
- Email must be unique within agency

---

## Case Management Context (DDD)

Complex domain for cross-agency case coordination.

### Case (Aggregate Root)

```
┌─────────────────────────────────────────┐
│             CASE (Aggregate)             │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  case_number     : string (unique)       │  ← Human-readable
│  type            : CaseType              │
│  status          : CaseStatus            │
│  priority        : Priority              │
│  title           : string                │
│  description     : text                  │
│  owning_agency   : UUID (FK)             │
│  lead_worker     : UUID (FK)             │
│                                          │
│  # Participants (embedded)               │
│  participants    : []Participant         │
│                                          │
│  # Assignments (embedded)                │
│  assignments     : []Assignment          │
│                                          │
│  # Timeline (embedded)                   │
│  events          : []CaseEvent           │
│                                          │
│  # SLA tracking                          │
│  sla_deadline    : timestamp             │
│  sla_status      : SLAStatus             │
│                                          │
│  # Cross-agency                          │
│  shared_with     : []UUID (agency IDs)   │
│  access_level    : map[UUID]AccessLevel  │
│                                          │
│  created_at      : timestamp             │
│  updated_at      : timestamp             │
│  closed_at       : timestamp (nullable)  │
└─────────────────────────────────────────┘

CaseType: enum {
  CHILD_WELFARE
  CRIMINAL
  ADMINISTRATIVE
  HEALTHCARE
  SOCIAL_ASSISTANCE
  TAX
  CIVIL
}

CaseStatus: enum {
  DRAFT
  OPEN
  IN_PROGRESS
  PENDING_TRANSFER
  PENDING_DOCUMENTS
  UNDER_REVIEW
  ESCALATED
  CLOSED
  ARCHIVED
}

Priority: enum {
  LOW
  MEDIUM
  HIGH
  URGENT
  EMERGENCY
}

SLAStatus: enum {
  ON_TRACK
  AT_RISK      ← 75% of time elapsed
  BREACHED
  PAUSED       ← Waiting on external
}
```

---

### Participant (Entity within Case)

```
┌─────────────────────────────────────────┐
│             PARTICIPANT                  │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  case_id         : UUID (FK)             │
│  citizen_id      : UUID (FK, nullable)   │
│  role            : ParticipantRole       │
│  name            : string                │  ← Denormalized for perf
│  contact         : ContactInfo           │
│  notes           : text                  │
│  added_at        : timestamp             │
│  added_by        : UUID                  │
└─────────────────────────────────────────┘

ParticipantRole: enum {
  APPLICANT        ← Person who initiated
  SUBJECT          ← Person case is about
  GUARDIAN         ← Legal guardian
  REPRESENTATIVE   ← Lawyer, advocate
  WITNESS
  EXPERT           ← Medical, forensic
  OTHER
}
```

---

### Assignment (Entity within Case)

```
┌─────────────────────────────────────────┐
│              ASSIGNMENT                  │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  case_id         : UUID (FK)             │
│  agency_id       : UUID (FK)             │
│  worker_id       : UUID (FK)             │
│  role            : AssignmentRole        │
│  status          : AssignmentStatus      │
│  assigned_at     : timestamp             │
│  assigned_by     : UUID                  │
│  completed_at    : timestamp (nullable)  │
│  notes           : text                  │
└─────────────────────────────────────────┘

AssignmentRole: enum {
  LEAD             ← Primary responsible
  SUPPORT          ← Assisting
  REVIEWER         ← Approval authority
  OBSERVER         ← Read-only access
}

AssignmentStatus: enum {
  ACTIVE
  COMPLETED
  REASSIGNED
  DECLINED
}
```

---

### CaseEvent (Entity within Case)

```
┌─────────────────────────────────────────┐
│              CASE_EVENT                  │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  case_id         : UUID (FK)             │
│  type            : CaseEventType         │
│  actor_id        : UUID                  │  ← Worker who did it
│  actor_agency    : UUID                  │
│  description     : string                │
│  data            : JSONB                 │  ← Event-specific payload
│  timestamp       : timestamp             │
└─────────────────────────────────────────┘

CaseEventType: enum {
  CREATED
  UPDATED
  STATUS_CHANGED
  ASSIGNED
  REASSIGNED
  TRANSFERRED
  ESCALATED
  DOCUMENT_ADDED
  DOCUMENT_SIGNED
  NOTE_ADDED
  PARTICIPANT_ADDED
  SHARED
  ACCESS_CHANGED
  SLA_WARNING
  SLA_BREACHED
  CLOSED
  REOPENED
}
```

**Case Business Rules:**

1. **Ownership**: Case must have exactly one owning agency
2. **Lead Worker**: Case must have exactly one lead worker from owning agency
3. **Transfer**: When transferred, previous agency retains read access
4. **SLA**: SLA deadline calculated based on case type and priority
5. **Closure**: Cannot close with pending assignments
6. **Sharing**: Access level per agency (none/read/write/full)
7. **Audit**: All changes recorded as CaseEvents

---

## Dispatch Context (DDD)

Emergency response coordination.

### Incident (Aggregate Root)

```
┌─────────────────────────────────────────┐
│           INCIDENT (Aggregate)           │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  incident_number : string (unique)       │
│  type            : IncidentType          │
│  severity        : Severity              │
│  status          : IncidentStatus        │
│  title           : string                │
│  description     : text                  │
│  location        : Location              │
│  reported_by     : ReporterInfo          │
│  reported_at     : timestamp             │
│                                          │
│  # Dispatches (embedded)                 │
│  dispatches      : []Dispatch            │
│                                          │
│  # Related case (if created)             │
│  case_id         : UUID (nullable)       │
│                                          │
│  created_at      : timestamp             │
│  resolved_at     : timestamp (nullable)  │
└─────────────────────────────────────────┘

IncidentType: enum {
  CRIME_IN_PROGRESS
  CRIME_REPORTED
  MEDICAL_EMERGENCY
  FIRE
  ACCIDENT_TRAFFIC
  ACCIDENT_WORKPLACE
  DOMESTIC_VIOLENCE
  CHILD_ENDANGERMENT
  MISSING_PERSON
  NATURAL_DISASTER
  PUBLIC_DISTURBANCE
  OTHER
}

Severity: enum {
  LOW
  MEDIUM
  HIGH
  CRITICAL
}

IncidentStatus: enum {
  REPORTED
  DISPATCHED
  ON_SCENE
  IN_PROGRESS
  RESOLVED
  CLOSED
  CANCELLED
}

Location {
  address     : string
  city        : string
  coordinates : Point (lat, lng)
  accuracy    : float           ← meters
  description : string          ← "Near the blue building"
}

ReporterInfo {
  name        : string
  phone       : string
  is_anonymous: boolean
  is_victim   : boolean
}
```

---

### Unit

```
┌─────────────────────────────────────────┐
│                 UNIT                     │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  agency_id       : UUID (FK)             │
│  call_sign       : string (unique)       │  ← e.g., "PATROL-23"
│  type            : UnitType              │
│  status          : UnitStatus            │
│  capabilities    : []string              │
│  crew            : []UUID (worker IDs)   │
│  vehicle_plate   : string                │
│  current_location: Point                 │
│  location_updated: timestamp             │
│  home_base       : Point                 │
│  created_at      : timestamp             │
└─────────────────────────────────────────┘

UnitType: enum {
  POLICE_PATROL
  POLICE_DETECTIVE
  POLICE_SWAT
  AMBULANCE_BLS      ← Basic Life Support
  AMBULANCE_ALS      ← Advanced Life Support
  FIRE_ENGINE
  FIRE_LADDER
  SOCIAL_WORKER
  INSPECTOR
}

UnitStatus: enum {
  AVAILABLE
  DISPATCHED
  EN_ROUTE
  ON_SCENE
  BUSY
  OFF_DUTY
  OUT_OF_SERVICE
}
```

---

### Dispatch (Entity within Incident)

```
┌─────────────────────────────────────────┐
│               DISPATCH                   │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  incident_id     : UUID (FK)             │
│  unit_id         : UUID (FK)             │
│  status          : DispatchStatus        │
│  priority        : int                   │  ← Order of dispatch
│  dispatched_at   : timestamp             │
│  dispatched_by   : UUID                  │
│  acknowledged_at : timestamp (nullable)  │
│  en_route_at     : timestamp (nullable)  │
│  on_scene_at     : timestamp (nullable)  │
│  cleared_at      : timestamp (nullable)  │
│  notes           : text                  │
└─────────────────────────────────────────┘

DispatchStatus: enum {
  PENDING          ← Waiting for ack
  ACKNOWLEDGED
  EN_ROUTE
  ON_SCENE
  COMPLETED
  CANCELLED
  NO_RESPONSE
}
```

**Dispatch Business Rules:**

1. **Unit Availability**: Can only dispatch available units
2. **Proximity**: System suggests nearest available units
3. **Capabilities**: Unit must have required capabilities for incident type
4. **Multi-agency**: Incident can have dispatches from multiple agencies
5. **Escalation**: Auto-escalate if no acknowledgment within 2 minutes
6. **Case Creation**: Certain incident types auto-create cases

---

## Document Context (DDD-lite)

Document lifecycle and signatures.

### Document (Aggregate Root)

```
┌─────────────────────────────────────────┐
│              DOCUMENT                    │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  document_number : string (unique)       │
│  type            : DocumentType          │
│  status          : DocumentStatus        │
│  title           : string                │
│  description     : text                  │
│                                          │
│  # Ownership                             │
│  owner_agency    : UUID (FK)             │
│  created_by      : UUID (FK)             │
│                                          │
│  # References                            │
│  case_id         : UUID (nullable)       │
│  incident_id     : UUID (nullable)       │
│                                          │
│  # Current version                       │
│  current_version : int                   │
│  versions        : []DocumentVersion    │
│                                          │
│  # Signatures                            │
│  signatures      : []Signature           │
│  requires_sig    : []UUID (worker IDs)   │
│                                          │
│  # Access                                │
│  shared_with     : []UUID (agency IDs)   │
│                                          │
│  created_at      : timestamp             │
│  updated_at      : timestamp             │
└─────────────────────────────────────────┘

DocumentType: enum {
  REPORT
  STATEMENT
  DECISION
  CERTIFICATE
  EVIDENCE
  FORM
  CORRESPONDENCE
  CONTRACT
  OTHER
}

DocumentStatus: enum {
  DRAFT
  PENDING_SIGNATURE
  PARTIALLY_SIGNED
  SIGNED
  REJECTED
  ARCHIVED
  VOID
}
```

---

### DocumentVersion

```
┌─────────────────────────────────────────┐
│           DOCUMENT_VERSION               │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  document_id     : UUID (FK)             │
│  version         : int                   │
│  file_path       : string                │  ← MinIO path
│  file_hash       : string (SHA-256)      │
│  file_size       : int (bytes)           │
│  mime_type       : string                │
│  created_at      : timestamp             │
│  created_by      : UUID                  │
│  change_summary  : string                │
└─────────────────────────────────────────┘
```

---

### Signature

```
┌─────────────────────────────────────────┐
│              SIGNATURE                   │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  document_id     : UUID (FK)             │
│  version         : int                   │
│  signer_id       : UUID (FK)             │
│  signer_agency   : UUID (FK)             │
│  type            : SignatureType         │
│  status          : SignatureStatus       │
│  signature_data  : bytes                 │  ← PAdES/XAdES
│  certificate     : bytes                 │
│  timestamp       : timestamp             │
│  timestamp_token : bytes                 │  ← TSA token
│  reason          : string                │
│  location        : string                │
└─────────────────────────────────────────┘

SignatureType: enum {
  SIMPLE           ← Click to sign
  ADVANCED         ← Certificate-based
  QUALIFIED        ← QES (eIDAS)
}

SignatureStatus: enum {
  PENDING
  SIGNED
  REJECTED
  REVOKED
}
```

**Document Business Rules:**

1. **Immutable Versions**: Once created, version content cannot change
2. **Hash Verification**: File integrity verified by SHA-256
3. **Signature Order**: Some documents require sequential signatures
4. **QES Required**: Certain document types require qualified signatures
5. **Retention**: Documents retained per legal requirements

---

## Messaging Context (Simple)

Secure inter-agency messaging.

### Conversation

```
┌─────────────────────────────────────────┐
│            CONVERSATION                  │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  type            : ConversationType      │
│  title           : string (nullable)     │
│  participants    : []ConvParticipant     │
│  case_id         : UUID (nullable)       │  ← Linked to case
│  incident_id     : UUID (nullable)       │  ← Linked to incident
│  created_at      : timestamp             │
│  created_by      : UUID                  │
└─────────────────────────────────────────┘

ConversationType: enum {
  DIRECT           ← 1:1
  GROUP            ← Multiple workers
  AGENCY_CHANNEL   ← Agency-wide
  CASE_THREAD      ← Attached to case
}

ConvParticipant {
  worker_id   : UUID
  agency_id   : UUID
  joined_at   : timestamp
  role        : string  ← "admin", "member"
}
```

---

### Message

```
┌─────────────────────────────────────────┐
│               MESSAGE                    │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  conversation_id : UUID (FK)             │
│  sender_id       : UUID (FK)             │
│  sender_agency   : UUID (FK)             │
│  type            : MessageType           │
│  content         : text (encrypted)      │
│  attachments     : []Attachment          │
│  reply_to        : UUID (nullable)       │
│  sent_at         : timestamp             │
│  edited_at       : timestamp (nullable)  │
│  deleted_at      : timestamp (nullable)  │
└─────────────────────────────────────────┘

MessageType: enum {
  TEXT
  FILE
  LOCATION
  SYSTEM          ← "X joined", "Y left"
}

Attachment {
  id          : UUID
  file_path   : string
  file_name   : string
  file_size   : int
  mime_type   : string
}
```

---

## Audit Context (Append-only)

Immutable audit trail.

### AuditEntry

```
┌─────────────────────────────────────────┐
│             AUDIT_ENTRY                  │
├─────────────────────────────────────────┤
│  id              : UUID                  │
│  timestamp       : timestamp             │
│  sequence        : bigint (monotonic)    │
│  hash            : string                │  ← Chain hash
│  prev_hash       : string                │
│                                          │
│  # Who                                   │
│  actor_type      : ActorType             │
│  actor_id        : UUID                  │
│  actor_agency    : UUID (nullable)       │
│  actor_ip        : string                │
│  actor_device    : string                │
│                                          │
│  # What                                  │
│  action          : string                │
│  resource_type   : string                │
│  resource_id     : UUID                  │
│  changes         : JSONB                 │  ← Before/after
│                                          │
│  # Context                               │
│  correlation_id  : UUID                  │  ← Request trace
│  session_id      : UUID                  │
│  justification   : string (nullable)     │
└─────────────────────────────────────────┘

ActorType: enum {
  CITIZEN
  WORKER
  SYSTEM
  EXTERNAL        ← API client
}
```

**Audit Business Rules:**

1. **Immutable**: Entries can never be modified or deleted
2. **Hash Chain**: Each entry includes hash of previous (tamper-evident)
3. **Mandatory**: All state changes must be audited
4. **Retention**: 7+ years for government, 10+ for criminal justice

---

## Entity Relationships

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        ENTITY RELATIONSHIPS                              │
└─────────────────────────────────────────────────────────────────────────┘

Agency (1) ─────────< Worker (N)
Agency (1) ─────────< Unit (N)
Agency (1) ─────────< Case (N)         [owning agency]

Worker (1) ─────────< Case (N)         [lead worker]
Worker (1) ─────────< Assignment (N)
Worker (N) ─────────< Unit (N)         [crew members]

Citizen (1) ────────< Participant (N)
Citizen (1) ────────< Worker (0..1)    [optional link]

Case (1) ───────────< Participant (N)
Case (1) ───────────< Assignment (N)
Case (1) ───────────< CaseEvent (N)
Case (1) ───────────< Document (N)
Case (0..1) ────────< Incident (N)

Incident (1) ───────< Dispatch (N)
Unit (1) ───────────< Dispatch (N)

Document (1) ───────< DocumentVersion (N)
Document (1) ───────< Signature (N)

Conversation (1) ───< Message (N)
Worker (N) ─────────< Conversation (N)  [participants]
```

---

## Database Schemas

Each module owns its schema:

```sql
-- Schemas (PostgreSQL)
CREATE SCHEMA identity;    -- Citizen, Agency, Worker
CREATE SCHEMA cases;       -- Case, Participant, Assignment, CaseEvent
CREATE SCHEMA dispatch;    -- Incident, Unit, Dispatch
CREATE SCHEMA documents;   -- Document, Version, Signature
CREATE SCHEMA messaging;   -- Conversation, Message
CREATE SCHEMA audit;       -- AuditEntry
```

**Cross-schema references** use UUIDs only (no foreign keys across schemas).

---

## Value Objects

Reusable value objects in shared kernel:

```go
// /internal/shared/types

type ID string           // UUID wrapper

type JMBG string         // Serbian personal ID (validated)

type Money struct {
    Amount   int64        // In smallest unit (paras)
    Currency string       // ISO 4217 (default: RSD)
}

type Address struct {
    Street     string
    City       string
    PostalCode string
    Country    string
    Location   *Point
}

type Point struct {
    Lat float64
    Lng float64
}

type ContactInfo struct {
    Email  string
    Phone  string
    Mobile string
}

type DateRange struct {
    From time.Time
    To   time.Time
}
```

---

## Invariants Summary

| Entity | Invariant |
|--------|-----------|
| Citizen | JMBG must be valid and unique |
| Agency | Code must be unique |
| Worker | Email unique within agency |
| Case | Must have owning agency and lead worker |
| Case | Cannot close with pending assignments |
| Incident | Can only dispatch available units |
| Unit | Call sign must be unique |
| Document | Versions are immutable once created |
| Signature | QES required for legal documents |
| Audit | Entries cannot be modified or deleted |
