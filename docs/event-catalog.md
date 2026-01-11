# Event Catalog

> Serbia Government Interoperability Platform

## Overview

This document defines all domain events, their publishers, subscribers, and payload schemas.

Events are the primary mechanism for cross-module communication in the modular monolith.

---

## Event Infrastructure

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         EVENT FLOW                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐     ┌──────────────┐     ┌──────────────┐                │
│  │  Module  │────►│  Event Bus   │────►│  Subscribers │                │
│  │(Publisher)│     │   (NATS)    │     │  (Handlers)  │                │
│  └──────────┘     └──────────────┘     └──────────────┘                │
│                          │                                              │
│                          ▼                                              │
│                   ┌──────────────┐                                     │
│                   │    Audit     │                                     │
│                   │   (Always)   │                                     │
│                   └──────────────┘                                     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Event Envelope

All events wrapped in standard envelope:

```go
type Event struct {
    // Metadata
    ID            string    `json:"id"`             // UUID
    Type          string    `json:"type"`           // e.g., "case.created"
    Source        string    `json:"source"`         // Module name
    Timestamp     time.Time `json:"timestamp"`
    CorrelationID string    `json:"correlation_id"` // Request trace

    // Actor
    ActorID       string    `json:"actor_id"`
    ActorType     string    `json:"actor_type"`     // citizen/worker/system
    ActorAgency   string    `json:"actor_agency,omitempty"`

    // Payload
    Data          any       `json:"data"`
}
```

---

## Event Naming Convention

```
{context}.{entity}.{action}

Examples:
- case.created
- case.status.changed
- case.transferred
- dispatch.incident.reported
- document.signed
```

---

## Case Events

### case.created

**Publisher:** Case Module
**Trigger:** New case created

```go
type CaseCreatedEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    Type         string    `json:"type"`
    Priority     string    `json:"priority"`
    Title        string    `json:"title"`
    OwningAgency string    `json:"owning_agency"`
    LeadWorker   string    `json:"lead_worker"`
    CreatedAt    time.Time `json:"created_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log creation |
| Workflow | Start case workflow |
| Notification | Notify assigned worker |

---

### case.status.changed

**Publisher:** Case Module
**Trigger:** Case status changes

```go
type CaseStatusChangedEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    OldStatus    string    `json:"old_status"`
    NewStatus    string    `json:"new_status"`
    Reason       string    `json:"reason,omitempty"`
    ChangedAt    time.Time `json:"changed_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log status change |
| Notification | Notify stakeholders |
| Workflow | Update workflow state |

---

### case.assigned

**Publisher:** Case Module
**Trigger:** Worker assigned to case

```go
type CaseAssignedEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    WorkerID     string    `json:"worker_id"`
    AgencyID     string    `json:"agency_id"`
    Role         string    `json:"role"`          // lead/support/reviewer
    AssignedBy   string    `json:"assigned_by"`
    AssignedAt   time.Time `json:"assigned_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log assignment |
| Notification | Notify assigned worker |
| Workflow | Update assignment tasks |

---

### case.transferred

**Publisher:** Case Module
**Trigger:** Case ownership transferred to another agency

```go
type CaseTransferredEvent struct {
    CaseID         string    `json:"case_id"`
    CaseNumber     string    `json:"case_number"`
    FromAgency     string    `json:"from_agency"`
    ToAgency       string    `json:"to_agency"`
    NewLeadWorker  string    `json:"new_lead_worker,omitempty"`
    Reason         string    `json:"reason"`
    TransferredAt  time.Time `json:"transferred_at"`
    TransferredBy  string    `json:"transferred_by"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log transfer |
| Notification | Notify both agencies |
| Workflow | Transfer workflow ownership |
| Federation | Sync to receiving agency gateway |

---

### case.escalated

**Publisher:** Case Module / Workflow
**Trigger:** Case escalated due to SLA or manual escalation

```go
type CaseEscalatedEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    Level        int       `json:"level"`         // 1-4
    Reason       string    `json:"reason"`        // sla_breach, manual, etc.
    EscalatedTo  string    `json:"escalated_to"`  // Worker or role
    EscalatedAt  time.Time `json:"escalated_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log escalation |
| Notification | Alert escalation target |
| Workflow | Trigger escalation workflow |

---

### case.sla.warning

**Publisher:** Workflow (SLA monitor)
**Trigger:** SLA approaching deadline (75%)

```go
type CaseSLAWarningEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    Deadline     time.Time `json:"deadline"`
    TimeLeft     string    `json:"time_left"`     // e.g., "2h 30m"
    PercentUsed  int       `json:"percent_used"`  // 75, 90, etc.
    WarningLevel string    `json:"warning_level"` // warning, critical
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log warning |
| Notification | Alert lead worker + supervisor |
| Case | Update SLA status |

---

### case.sla.breached

**Publisher:** Workflow (SLA monitor)
**Trigger:** SLA deadline passed

```go
type CaseSLABreachedEvent struct {
    CaseID        string    `json:"case_id"`
    CaseNumber    string    `json:"case_number"`
    Deadline      time.Time `json:"deadline"`
    BreachedAt    time.Time `json:"breached_at"`
    OverdueBy     string    `json:"overdue_by"`
    LeadWorker    string    `json:"lead_worker"`
    OwningAgency  string    `json:"owning_agency"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log breach |
| Notification | Alert chain of command |
| Case | Update SLA status, trigger escalation |

---

### case.closed

**Publisher:** Case Module
**Trigger:** Case closed

```go
type CaseClosedEvent struct {
    CaseID       string    `json:"case_id"`
    CaseNumber   string    `json:"case_number"`
    Resolution   string    `json:"resolution"`    // resolved, dismissed, etc.
    ClosedBy     string    `json:"closed_by"`
    ClosedAt     time.Time `json:"closed_at"`
    Summary      string    `json:"summary,omitempty"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log closure |
| Workflow | Complete workflow |
| Notification | Notify participants |
| Document | Archive related documents |

---

## Dispatch Events

### dispatch.incident.reported

**Publisher:** Dispatch Module
**Trigger:** New incident reported (call received, app submission)

```go
type IncidentReportedEvent struct {
    IncidentID     string    `json:"incident_id"`
    IncidentNumber string    `json:"incident_number"`
    Type           string    `json:"type"`
    Severity       string    `json:"severity"`
    Location       Location  `json:"location"`
    Description    string    `json:"description"`
    ReportedAt     time.Time `json:"reported_at"`
}

type Location struct {
    Address     string  `json:"address"`
    City        string  `json:"city"`
    Lat         float64 `json:"lat"`
    Lng         float64 `json:"lng"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log incident |
| Dispatch | Find nearest units |
| Notification | Alert dispatch center |

---

### dispatch.unit.dispatched

**Publisher:** Dispatch Module
**Trigger:** Unit dispatched to incident

```go
type UnitDispatchedEvent struct {
    DispatchID     string    `json:"dispatch_id"`
    IncidentID     string    `json:"incident_id"`
    IncidentNumber string    `json:"incident_number"`
    UnitID         string    `json:"unit_id"`
    UnitCallSign   string    `json:"unit_call_sign"`
    UnitType       string    `json:"unit_type"`
    DispatchedAt   time.Time `json:"dispatched_at"`
    DispatchedBy   string    `json:"dispatched_by"`
    ETA            string    `json:"eta,omitempty"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log dispatch |
| Notification | Notify unit (mobile push) |
| Dispatch | Update unit status |

---

### dispatch.unit.status.changed

**Publisher:** Dispatch Module (via mobile app)
**Trigger:** Unit status changes (acknowledged, en_route, on_scene, etc.)

```go
type UnitStatusChangedEvent struct {
    UnitID       string    `json:"unit_id"`
    UnitCallSign string    `json:"unit_call_sign"`
    IncidentID   string    `json:"incident_id,omitempty"`
    OldStatus    string    `json:"old_status"`
    NewStatus    string    `json:"new_status"`
    Location     *Location `json:"location,omitempty"`
    ChangedAt    time.Time `json:"changed_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log status change |
| Dispatch | Update dashboard |
| Notification | Notify dispatch center |

---

### dispatch.unit.location.updated

**Publisher:** Dispatch Module (via mobile app GPS)
**Trigger:** Unit reports location (every 30s when active)

```go
type UnitLocationUpdatedEvent struct {
    UnitID       string    `json:"unit_id"`
    UnitCallSign string    `json:"unit_call_sign"`
    Location     Location  `json:"location"`
    Speed        float64   `json:"speed,omitempty"`      // km/h
    Heading      float64   `json:"heading,omitempty"`    // degrees
    Accuracy     float64   `json:"accuracy"`             // meters
    Timestamp    time.Time `json:"timestamp"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Dispatch | Update map, recalculate ETAs |

> Note: High-frequency event, not persisted to audit log.

---

### dispatch.incident.resolved

**Publisher:** Dispatch Module
**Trigger:** Incident resolved

```go
type IncidentResolvedEvent struct {
    IncidentID     string    `json:"incident_id"`
    IncidentNumber string    `json:"incident_number"`
    Resolution     string    `json:"resolution"`
    ResolvedAt     time.Time `json:"resolved_at"`
    ResolvedBy     string    `json:"resolved_by"`
    CaseCreated    bool      `json:"case_created"`
    CaseID         string    `json:"case_id,omitempty"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log resolution |
| Dispatch | Release units |
| Case | Create follow-up case if needed |
| Notification | Notify stakeholders |

---

## Document Events

### document.created

**Publisher:** Document Module
**Trigger:** New document created

```go
type DocumentCreatedEvent struct {
    DocumentID     string    `json:"document_id"`
    DocumentNumber string    `json:"document_number"`
    Type           string    `json:"type"`
    Title          string    `json:"title"`
    OwnerAgency    string    `json:"owner_agency"`
    CreatedBy      string    `json:"created_by"`
    CaseID         string    `json:"case_id,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log creation |
| Case | Update case timeline |

---

### document.version.added

**Publisher:** Document Module
**Trigger:** New version uploaded

```go
type DocumentVersionAddedEvent struct {
    DocumentID     string    `json:"document_id"`
    DocumentNumber string    `json:"document_number"`
    Version        int       `json:"version"`
    FileHash       string    `json:"file_hash"`
    FileSize       int64     `json:"file_size"`
    AddedBy        string    `json:"added_by"`
    AddedAt        time.Time `json:"added_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log version |
| Notification | Notify document subscribers |

---

### document.signature.requested

**Publisher:** Document Module
**Trigger:** Signature requested from worker

```go
type SignatureRequestedEvent struct {
    DocumentID     string    `json:"document_id"`
    DocumentNumber string    `json:"document_number"`
    SignerID       string    `json:"signer_id"`
    SignerAgency   string    `json:"signer_agency"`
    SignatureType  string    `json:"signature_type"` // simple/advanced/qualified
    RequestedBy    string    `json:"requested_by"`
    RequestedAt    time.Time `json:"requested_at"`
    Deadline       time.Time `json:"deadline,omitempty"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log request |
| Notification | Notify signer |
| Workflow | Track signature deadline |

---

### document.signed

**Publisher:** Document Module
**Trigger:** Document signed

```go
type DocumentSignedEvent struct {
    DocumentID     string    `json:"document_id"`
    DocumentNumber string    `json:"document_number"`
    Version        int       `json:"version"`
    SignerID       string    `json:"signer_id"`
    SignerAgency   string    `json:"signer_agency"`
    SignatureType  string    `json:"signature_type"`
    SignedAt       time.Time `json:"signed_at"`
    AllSigned      bool      `json:"all_signed"` // All required signatures done
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log signature |
| Document | Update status if all signed |
| Case | Update case timeline |
| Notification | Notify requester |

---

### document.signature.rejected

**Publisher:** Document Module
**Trigger:** Signer rejected signing request

```go
type SignatureRejectedEvent struct {
    DocumentID     string    `json:"document_id"`
    DocumentNumber string    `json:"document_number"`
    SignerID       string    `json:"signer_id"`
    SignerAgency   string    `json:"signer_agency"`
    Reason         string    `json:"reason"`
    RejectedAt     time.Time `json:"rejected_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log rejection |
| Document | Update status |
| Notification | Notify document owner |
| Workflow | Handle rejection |

---

## Messaging Events

### messaging.message.sent

**Publisher:** Messaging Module
**Trigger:** Message sent in conversation

```go
type MessageSentEvent struct {
    MessageID      string    `json:"message_id"`
    ConversationID string    `json:"conversation_id"`
    SenderID       string    `json:"sender_id"`
    SenderAgency   string    `json:"sender_agency"`
    Type           string    `json:"type"` // text/file/location
    SentAt         time.Time `json:"sent_at"`
    // Content not included (E2EE)
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log message metadata (not content) |
| Notification | Push to recipients |

---

## Citizen Events

### citizen.registered

**Publisher:** Citizen Module
**Trigger:** New citizen registered

```go
type CitizenRegisteredEvent struct {
    CitizenID    string    `json:"citizen_id"`
    JMBG         string    `json:"jmbg"`        // Masked: ******1234
    EIDLinked    bool      `json:"eid_linked"`
    RegisteredAt time.Time `json:"registered_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log registration |

---

### citizen.eid.linked

**Publisher:** Citizen Module
**Trigger:** Citizen linked Serbia eID

```go
type CitizenEIDLinkedEvent struct {
    CitizenID      string    `json:"citizen_id"`
    AssuranceLevel string    `json:"assurance_level"` // basic/high/highest
    LinkedAt       time.Time `json:"linked_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log eID link |
| Notification | Confirm to citizen |

---

## Agency Events

### agency.worker.added

**Publisher:** Agency Module
**Trigger:** New worker added to agency

```go
type WorkerAddedEvent struct {
    WorkerID   string    `json:"worker_id"`
    AgencyID   string    `json:"agency_id"`
    EmployeeID string    `json:"employee_id"`
    Roles      []string  `json:"roles"`
    AddedAt    time.Time `json:"added_at"`
    AddedBy    string    `json:"added_by"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log addition |

---

### agency.worker.role.changed

**Publisher:** Agency Module
**Trigger:** Worker roles modified

```go
type WorkerRoleChangedEvent struct {
    WorkerID   string    `json:"worker_id"`
    AgencyID   string    `json:"agency_id"`
    OldRoles   []string  `json:"old_roles"`
    NewRoles   []string  `json:"new_roles"`
    ChangedBy  string    `json:"changed_by"`
    ChangedAt  time.Time `json:"changed_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log role change |
| Auth | Invalidate cached permissions |

---

## Federation Events

### federation.agency.connected

**Publisher:** Federation (Gateway)
**Trigger:** Agency gateway connected to federation

```go
type AgencyConnectedEvent struct {
    AgencyID      string    `json:"agency_id"`
    AgencyCode    string    `json:"agency_code"`
    GatewayID     string    `json:"gateway_id"`
    CertificateID string    `json:"certificate_id"`
    ConnectedAt   time.Time `json:"connected_at"`
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log connection |
| Agency | Update agency status |

---

### federation.request.received

**Publisher:** Federation (Gateway)
**Trigger:** Cross-agency request received

```go
type FederationRequestEvent struct {
    RequestID     string    `json:"request_id"`
    FromAgency    string    `json:"from_agency"`
    ToAgency      string    `json:"to_agency"`
    Service       string    `json:"service"`
    Method        string    `json:"method"`
    Timestamp     time.Time `json:"timestamp"`
    // Payload not included (may be sensitive)
}
```

**Subscribers:**
| Module | Action |
|--------|--------|
| Audit | Log request |

---

## Event Subscription Matrix

| Event | Audit | Case | Dispatch | Document | Notification | Workflow | Federation |
|-------|-------|------|----------|----------|--------------|----------|------------|
| case.created | ✓ | | | | ✓ | ✓ | |
| case.status.changed | ✓ | | | | ✓ | ✓ | |
| case.assigned | ✓ | | | | ✓ | ✓ | |
| case.transferred | ✓ | | | | ✓ | ✓ | ✓ |
| case.escalated | ✓ | | | | ✓ | ✓ | |
| case.sla.warning | ✓ | ✓ | | | ✓ | | |
| case.sla.breached | ✓ | ✓ | | | ✓ | | |
| case.closed | ✓ | | | ✓ | ✓ | ✓ | |
| dispatch.incident.reported | ✓ | | ✓ | | ✓ | | |
| dispatch.unit.dispatched | ✓ | | ✓ | | ✓ | | |
| dispatch.unit.status.changed | ✓ | | ✓ | | ✓ | | |
| dispatch.incident.resolved | ✓ | ✓ | ✓ | | ✓ | | |
| document.created | ✓ | ✓ | | | | | |
| document.signed | ✓ | ✓ | | ✓ | ✓ | | |
| messaging.message.sent | ✓ | | | | ✓ | | |

---

## NATS Subjects

```
# Subject naming: {context}.{entity}.{action}

# Case events
gov.case.created
gov.case.status.changed
gov.case.assigned
gov.case.transferred
gov.case.escalated
gov.case.sla.warning
gov.case.sla.breached
gov.case.closed

# Dispatch events
gov.dispatch.incident.reported
gov.dispatch.unit.dispatched
gov.dispatch.unit.status.changed
gov.dispatch.unit.location.updated
gov.dispatch.incident.resolved

# Document events
gov.document.created
gov.document.version.added
gov.document.signature.requested
gov.document.signed
gov.document.signature.rejected

# Messaging events
gov.messaging.message.sent

# Citizen events
gov.citizen.registered
gov.citizen.eid.linked

# Agency events
gov.agency.worker.added
gov.agency.worker.role.changed

# Federation events
gov.federation.agency.connected
gov.federation.request.received
```

---

## AsyncAPI Specification

Events documented in AsyncAPI format at: `/api/asyncapi/events.yaml`

```yaml
asyncapi: 3.0.0
info:
  title: Serbia Gov Platform Events
  version: 1.0.0

servers:
  production:
    host: nats.internal:4222
    protocol: nats

channels:
  caseCreated:
    address: gov.case.created
    messages:
      caseCreated:
        $ref: '#/components/messages/CaseCreated'
  # ... etc
```
