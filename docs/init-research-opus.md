# Technical Blueprint for Serbia's Government Interoperability Platform

**Serbia can build a unified public services coordination system using proven open-source technologies that already connect 929+ institutions across Estonia.** The recommended architecture centers on X-Road as the data exchange backbone, combined with event-driven microservices for real-time coordination and modern case management workflows. This specification provides the technical foundation for an RFP that will enable police, healthcare, social services, courts, and local government to share data securely while respecting EU standards critical for accession.

The platform addresses two core functions: **real-time operational coordination** (duty rosters, emergency dispatch, escalation procedures) and **cross-institutional case management** (citizen cases flowing through multiple agencies). Implementation follows a phased approach validated by Estonia (10+ years), Finland, Denmark, and Ukraine—countries that transformed government efficiency while maintaining data sovereignty.

---

## X-Road as the interoperability foundation

**X-Road provides the most battle-tested, open-source government data exchange layer available**, with MIT licensing ensuring full sovereignty. Developed in Estonia since 2001 and now deployed in **25+ countries**, it enables secure, decentralized data sharing without creating a central database—a critical feature for multi-agency trust.

### Core architecture principles

X-Road operates on a **distributed peer-to-peer model** where each participating organization maintains a Security Server that handles authentication, encryption, digital signing, and comprehensive logging. No central database exists; data remains with its original holders, eliminating single points of failure and concentrating risk.

| Component | Function | Deployment |
|-----------|----------|------------|
| **Central Server** | Trust management, member registry, service catalog | Operated by national authority |
| **Security Server** | Per-organization gateway; handles mTLS, signing, logging | Each agency deploys own instance |
| **Configuration Proxy** | Distributes trust updates across network | High-availability cluster |

**Technical specifications** include REST API support (OpenAPI 3.0), SOAP/WSDL for legacy systems, TLS 1.2+ with mutual authentication, batch digital signatures, and eIDAS compliance pathways. The current stable version (7.6.x) runs on Linux (Ubuntu/RHEL), Docker containers, or Kubernetes, with PostgreSQL as the database.

### Proven scale and reliability

Estonia's X-Road now connects **929 institutions, 2,800+ services, and 1,887 information systems** with virtually zero downtime since 2001. The Estonia-Finland federation demonstrates cross-border capability essential for Serbia's EU aspirations. Ukraine's Trembita platform—built on X-Road—maintained operations throughout active conflict, demonstrating exceptional resilience.

---

## EU compliance framework for accession readiness

As an EU candidate country, Serbia must align with the **European Interoperability Framework (EIF)** and prepare for the **Interoperable Europe Act** (Regulation EU 2024/903), which entered force on July 12, 2024. Compliance enables future participation in trans-European digital services and demonstrates governance maturity.

### Four interoperability layers

The EIF mandates coordination across legal, organizational, semantic, and technical dimensions:

- **Legal interoperability**: Perform interoperability checks when drafting legislation; ensure data protection alignment with GDPR-equivalent frameworks
- **Organizational interoperability**: Document business processes; establish Service Level Agreements between agencies; define governance arrangements
- **Semantic interoperability**: Adopt SEMIC Core Vocabularies (Person, Business, Location, Public Service); implement controlled vocabularies and taxonomies
- **Technical interoperability**: Standard communication protocols; interface specifications following EIRA Architecture Building Blocks

### Critical EU building blocks to implement

| Building Block | Purpose | Technical Standard |
|----------------|---------|-------------------|
| **eDelivery** | Secure cross-border data exchange | OASIS ebMS3/AS4 profile via Domibus |
| **eID** | Electronic identification federation | eIDAS node with SAML 2.0/OIDC |
| **eSignature** | Document authenticity | XAdES, PAdES, CAdES formats |
| **OOTS** | Once-Only Technical System for evidence exchange | REST/JSON APIs connecting authentic sources |

The **Once-Only Technical System (OOTS)**, operational since December 2023, enables citizens to authorize cross-border evidence sharing. Serbia should implement an eDelivery Access Point and Data Service connectors to authentic sources (population registry, tax records, educational credentials) in preparation for eventual OOTS integration.

---

## Security architecture meeting zero-trust requirements

The platform must implement **zero-trust principles** per NIST SP 800-207 while ensuring GDPR-equivalent data protection for sensitive inter-agency exchanges involving police, healthcare, and social services data.

### Zero-trust implementation

The core tenet—**"never trust, always verify"**—requires seven operational principles: treating all resources as untrusted, securing all communications regardless of network location, granting access per-session based on dynamic policy evaluation, continuous asset monitoring, strict authentication enforcement, and maximum telemetry collection.

**Implementation components include:**

| Component | Function | Recommendation |
|-----------|----------|----------------|
| Policy Engine (PE) | Access decisions based on policy | Open Policy Agent (OPA) |
| Policy Enforcement Point (PEP) | Connection control | Service mesh sidecar proxies |
| Identity-centric controls | Continuous authentication | Keycloak + MFA enforcement |
| Micro-segmentation | Network isolation | Kubernetes network policies + service mesh |

### Data protection for multi-agency exchange

Under GDPR Article 6, processing between public authorities requires explicit legal basis (legal obligation or public task) established in Member State law. The platform must implement:

- **Data minimization controls**: Share minimum data necessary for each transaction
- **Purpose limitation enforcement**: Technical controls preventing scope creep across agencies
- **Joint controller arrangements**: Documented responsibility allocation when agencies jointly determine processing purposes
- **Data subject rights portals**: Centralized mechanism for citizens to exercise access, rectification, and erasure rights across all participating agencies

**Access control** should combine RBAC (role-based) for organizational functions with ABAC (attribute-based) for fine-grained, cross-agency authorization using XACML 3.0 policies. Attributes include subject (role, clearance, citizenship), resource (classification, owner), action (read, write), and environment (time, location, threat level).

### Comprehensive audit trail requirements

Every transaction must be logged with: user identity, timestamp (cryptographic), action type, old/new values, business justification, source IP/device, and result. Logs require **write-once-read-many (WORM) storage** with cryptographic hash chains for integrity. Retention periods typically span 1-7 years for public sector, with criminal justice potentially requiring 10+ years.

---

## Technical architecture for scale and resilience

The recommended architecture uses **domain-driven microservices** with event-driven communication, enabling independent scaling of workloads while maintaining transactional consistency across agency boundaries.

### Service architecture

| Pattern | Application | Technology |
|---------|-------------|------------|
| API Gateway | Single entry point, rate limiting, auth | **Tyk** (complete lifecycle management) or Kong |
| Event Streaming | Audit logs, case events, notifications | **Apache Kafka** (millions/sec, full replay) |
| Task Queues | Document processing, batch operations | **RabbitMQ** (priority queuing) |
| Service Mesh | mTLS, traffic management, observability | **Linkerd** (simpler) or Istio (advanced) |
| Workflow Engine | BPMN 2.0 process orchestration | **Camunda 7** or **Flowable** |

**Kafka configuration for government** should use replication factor 3, min.insync.replicas 2, acks=all for strong durability, and 30+ day retention for audit compliance. Event schemas follow CloudEvents specification with government extensions (agencyId, caseId, correlationId, securityClassification).

### Legacy system integration strategy

Most Serbian government systems will require adaptation rather than replacement. The **Strangler Fig pattern** enables gradual modernization:

1. Deploy API façade in front of legacy system
2. Route all traffic through façade
3. Incrementally build microservices for new functionality
4. Redirect traffic endpoint-by-endpoint to new services
5. Decommission legacy components after validation

**Anti-corruption layers** protect new systems from legacy data models: a façade accepts modern requests, an adapter handles protocol conversion (SOAP→REST), and a translator transforms data structures. Apache Camel provides 200+ enterprise integration patterns for connecting heterogeneous systems.

### Scalability and disaster recovery

| Tier | System Type | RPO | RTO |
|------|-------------|-----|-----|
| **Critical** | Citizen services, case management | 15 minutes | 1 hour |
| **Standard** | Internal agency systems | 1 hour | 4 hours |
| **Non-critical** | Analytics, reporting | 24 hours | 24 hours |

PostgreSQL high-availability uses Patroni for automatic failover with synchronous replication across availability zones. Active-active deployment across datacenters enables sub-second failover for critical citizen-facing services.

---

## Real-time coordination capabilities

The platform's operational coordination layer manages duty rosters, emergency dispatch, inter-agency communication, and escalation workflows across all participating agencies.

### Emergency dispatch coordination

For multi-agency emergency response (police, ambulance, fire, social services), implement a Computer-Aided Dispatch (CAD) system following **NENA i3 standards** for interoperability:

- **ESInet (Emergency Services IP Network)**: Shared IP infrastructure across agencies
- **EIDO (Emergency Incident Data Object)**: Standard format for incident data exchange
- **Real-time AVL**: GPS tracking of all response units via WebSocket connections
- **Automatic dispatch recommendations**: Based on unit proximity, capability, and availability

Open-source options include **Resgrid Core** (Apache 2.0, full CAD with iOS/Android apps) and **Tickets CAD** (PHP/MySQL, Google Maps integration).

### Secure inter-agency messaging

**Matrix protocol** (via Element) provides government-grade secure messaging, already deployed by the French government (300,000+ civil servants), German Bundeswehr, and NATO. Key features include:

- Decentralized federated architecture maintaining data sovereignty
- End-to-end encryption (Olm for 1:1, Megolm for groups)
- Voice/video calling
- Bridges to existing platforms (SMS, email)
- Air-gapped deployment capability

### Escalation workflow architecture

Implement multi-tier escalation using BPMN 2.0 workflow engines:

```
Level 1 (50% SLA elapsed): Email notification to handler
Level 2 (75% SLA elapsed): SMS + dashboard alert to supervisor  
Level 3 (SLA breach imminent): Page on-call + cross-agency escalation
Level 4 (SLA breached): Executive notification + auto-reassignment
```

**Flowable** or **Camunda 7** provide human task management, decision tables (DMN), and process monitoring dashboards. SLA timers track handoff delays between agencies, triggering predictive alerts before breaches occur.

---

## Cross-institutional case management

The case management layer enables citizens' cases to flow seamlessly across agencies with full audit trails, document sharing, and privacy-preserving access controls.

### Unified case data model

```json
{
  "case_id": "UUID (cross-agency identifier)",
  "agency_references": [{"agency": "MUP", "local_id": "MUP-2026-001"}],
  "case_type": "child_welfare | criminal | administrative | healthcare",
  "status": "open | pending_transfer | awaiting_documents | closed",
  "participants": [
    {"role": "applicant", "person_id": "UUID", "agency": "CSW_Belgrade"},
    {"role": "case_worker", "person_id": "UUID", "agency": "Police_Novi_Sad"}
  ],
  "timeline": [{"event": "Case created", "timestamp": "ISO8601", "actor": "UUID"}],
  "permissions": {"agency_access": ["MUP", "CSW", "Health_Ministry"]}
}
```

Event sourcing captures all state changes as immutable events, enabling complete audit trails, state reconstruction at any point, and temporal queries ("citizen status as of specific date"). The CQRS pattern separates write operations from optimized read views (cases_by_citizen, cases_by_agency, cases_pending_action).

### Document management with digital signatures

**Alfresco Community** (LGPL) or **Nuxeo** (LGPL) provide enterprise content management with:

- Version control with major/minor versioning
- PDF/A for archival, ODF for editing
- Dublin Core metadata standards
- eIDAS-compliant digital signatures (QES for legal equivalence)

Qualified Electronic Signatures (QES) using qualified certificates provide legal equivalence to handwritten signatures under eIDAS. Implementation requires PAdES for PDF documents, XAdES for XML data exchanges.

---

## Implementation roadmap and governance

Based on successful implementations in Estonia, Finland, and Ukraine, Serbia should follow a phased approach spanning **4-5 years** for full ecosystem maturity.

### Phased implementation

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Foundation** | Year 1 | Legal framework, governance structure, X-Road pilot with 3-5 agencies |
| **Core Services** | Years 2-3 | 20-30 agencies connected, citizen portal launch, digital identity |
| **Scaling** | Years 3-4 | Full public sector rollout, private sector integration |
| **Maturation** | Years 4-5 | Advanced features (AI, proactive services), regional federation |

### Governance structure

Successful models consistently feature a **central digital authority** with clear mandate. The recommended structure:

1. **National Digital Authority**: Operational responsibility (modeled on Estonia's RIA)
2. **Cross-ministerial coordination board**: Policy alignment
3. **Technical standards committee**: Architecture decisions
4. **Public-private partnership council**: Private sector integration
5. **Citizen advisory group**: User-centered design validation

### Critical success factors from case studies

Every successful implementation shares these elements: **senior political sponsorship**, legal framework mandating participation, distributed architecture maintaining agency data ownership, security-by-design from inception, open standards avoiding vendor lock-in, phased rollout proving value early, user-centric design, and **10+ year commitment**.

Estonia's experience demonstrates that organizational preparation—defining data ownership, access rights, and governance—requires as much effort as technical implementation. Ukraine's Diia platform shows that strong political leadership can accelerate timelines dramatically, achieving 19 million users within 3 years.

---

## Recommended technology stack summary

| Layer | Technology | License | Justification |
|-------|------------|---------|---------------|
| **Interoperability** | X-Road | MIT | Proven at national scale, EU-aligned |
| **Identity** | Keycloak + national eID | Apache 2.0 | OAuth2/OIDC/SAML federation |
| **API Gateway** | Tyk | Open Source | Complete API lifecycle management |
| **Messaging** | Apache Kafka | Apache 2.0 | Event sourcing, audit logs |
| **Workflow** | Camunda 7 / Flowable | Apache 2.0 | BPMN 2.0, human tasks |
| **Documents** | Alfresco Community | LGPL | Enterprise content management |
| **Database** | PostgreSQL | PostgreSQL | Relational data, JSON support |
| **Search** | Elasticsearch | SSPL | Full-text case search |
| **Cache** | Redis | BSD | Session management |
| **Communication** | Matrix/Element | Apache 2.0 | Secure inter-agency messaging |
| **Monitoring** | Grafana + Prometheus | AGPL/Apache | Dashboards, alerting |
| **Service Mesh** | Linkerd | Apache 2.0 | mTLS, observability |

### Immediate next steps

1. **Engage NIIS** (Nordic Institute for Interoperability Solutions) for X-Road technical guidance
2. **Contact e-Governance Academy** (Estonia) for implementation training
3. **Develop legislative framework** enabling mandatory interoperability
4. **Identify pilot agencies** with strongest existing IT capability (recommend: population registry, tax administration, social services)
5. **Connect with Open Balkan partners** (Albania, North Macedonia) for regional alignment
6. **Apply for EU Digital Europe Programme** funding for technical infrastructure

This specification provides actionable technical requirements for an RFP that will deliver a secure, scalable, EU-compliant interoperability platform capable of transforming Serbia's public services coordination within a realistic 4-5 year timeline.