-- Event Store and Privacy tables for Serbia Government Interoperability Platform
-- Migration: 002_eventstore_privacy.sql

-----------------------------------------------------------
-- EVENTSTORE SCHEMA
-----------------------------------------------------------

CREATE SCHEMA IF NOT EXISTS eventstore;

-- Event store (append-only, immutable)
-- This is the source of truth for all domain events
CREATE TABLE eventstore.events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    version INT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data JSONB NOT NULL,
    metadata JSONB NOT NULL,

    -- Ensure ordering within aggregate (optimistic concurrency)
    UNIQUE(aggregate_id, version)
);

-- Optimized indexes for common queries
CREATE INDEX idx_events_aggregate ON eventstore.events(aggregate_id, version);
CREATE INDEX idx_events_type ON eventstore.events(event_type);
CREATE INDEX idx_events_timestamp ON eventstore.events(timestamp);
CREATE INDEX idx_events_aggregate_type ON eventstore.events(aggregate_type);

-- Global sequence for ordering across all events
CREATE SEQUENCE eventstore.global_sequence;

-- Snapshots for faster state rebuilding
CREATE TABLE eventstore.snapshots (
    aggregate_id UUID PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,
    version INT NOT NULL,
    state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snapshots_type ON eventstore.snapshots(aggregate_type);

-- Projections tracking (read model progress)
CREATE TABLE eventstore.projections (
    name VARCHAR(100) PRIMARY KEY,
    last_processed_sequence BIGINT NOT NULL DEFAULT 0,
    last_processed_event_id UUID,
    last_processed_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'running', -- running, paused, failed
    error_message TEXT
);

-- Prevent modifications to events (append-only)
CREATE OR REPLACE FUNCTION eventstore.prevent_event_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Events cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER events_no_update
    BEFORE UPDATE ON eventstore.events
    FOR EACH ROW
    EXECUTE FUNCTION eventstore.prevent_event_modification();

CREATE TRIGGER events_no_delete
    BEFORE DELETE ON eventstore.events
    FOR EACH ROW
    EXECUTE FUNCTION eventstore.prevent_event_modification();

-----------------------------------------------------------
-- PRIVACY SCHEMA
-----------------------------------------------------------

CREATE SCHEMA IF NOT EXISTS privacy;

-- Pseudonym mappings (LOCAL facility only - never replicated to central!)
-- This table stores the mapping between real JMBG and pseudonymized IDs
CREATE TABLE privacy.pseudonym_mappings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- We don't store JMBG directly, only a hash for lookup
    -- The actual JMBG is encrypted with the facility's HSM key
    jmbg_hash VARCHAR(64) NOT NULL,        -- SHA-256 of JMBG for lookup
    jmbg_encrypted BYTEA,                   -- Encrypted JMBG (optional, for recovery)

    -- The pseudonymized ID sent to central system
    pseudonym_id VARCHAR(48) NOT NULL UNIQUE,

    -- Facility identification
    facility_code VARCHAR(20) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique per facility - same JMBG can have different pseudonyms in different facilities
    UNIQUE(jmbg_hash, facility_code)
);

CREATE INDEX idx_pseudonym_jmbg_hash ON privacy.pseudonym_mappings(jmbg_hash);
CREATE INDEX idx_pseudonym_id ON privacy.pseudonym_mappings(pseudonym_id);
CREATE INDEX idx_pseudonym_facility ON privacy.pseudonym_mappings(facility_code);

-- Depseudonymization requests
-- Tracks all requests to reveal real identity from pseudonym
CREATE TABLE privacy.depseudonymization_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pseudonym_id VARCHAR(48) NOT NULL,

    -- Who is requesting
    requestor_id UUID NOT NULL,
    requestor_agency_id UUID NOT NULL,

    -- Why they need access
    purpose TEXT NOT NULL,
    legal_basis VARCHAR(50) NOT NULL, -- court_order, life_threat, child_protection, law_enforcement, subject_consent
    justification TEXT NOT NULL,

    -- Timing
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    -- Approval workflow
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, expired, revoked
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,

    -- Related case (if applicable)
    case_id UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_depseudo_requests_status ON privacy.depseudonymization_requests(status);
CREATE INDEX idx_depseudo_requests_pseudonym ON privacy.depseudonymization_requests(pseudonym_id);
CREATE INDEX idx_depseudo_requests_requestor ON privacy.depseudonymization_requests(requestor_id);
CREATE INDEX idx_depseudo_requests_expires ON privacy.depseudonymization_requests(expires_at);

-- Depseudonymization tokens
-- Time-limited tokens for approved access
CREATE TABLE privacy.depseudonymization_tokens (
    token VARCHAR(72) PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES privacy.depseudonymization_requests(id),
    pseudonym_id VARCHAR(48) NOT NULL,

    -- Validity
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,

    -- Usage tracking
    used_count INT NOT NULL DEFAULT 0,
    max_uses INT NOT NULL DEFAULT 3,
    last_used_at TIMESTAMPTZ,

    -- Revocation
    revoked_at TIMESTAMPTZ,
    revoked_by UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_depseudo_tokens_expires ON privacy.depseudonymization_tokens(expires_at);
CREATE INDEX idx_depseudo_tokens_pseudonym ON privacy.depseudonymization_tokens(pseudonym_id);

-- AI access requests
-- Tracks elevated access requests for AI systems
CREATE TABLE privacy.ai_access_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ai_system_id VARCHAR(100) NOT NULL,

    -- What level of access is requested
    requested_level INT NOT NULL, -- 0=aggregated, 1=pseudonymized, 2=linkable

    -- Justification
    purpose TEXT NOT NULL,
    data_scope JSONB, -- What data types, time ranges, etc.

    -- Who is requesting
    requested_by UUID NOT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Validity window
    expires_at TIMESTAMPTZ NOT NULL,

    -- Approval
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_by UUID,
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_access_system ON privacy.ai_access_requests(ai_system_id, status);
CREATE INDEX idx_ai_access_expires ON privacy.ai_access_requests(expires_at);

-- PII violation log
-- Records all detected attempts to leak personal data
CREATE TABLE privacy.pii_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- What was detected
    field VARCHAR(20) NOT NULL, -- jmbg, name, address, phone, email, lbo
    location TEXT NOT NULL,     -- API path, event type, etc.

    -- Who triggered it
    actor_id UUID,
    actor_agency_id UUID,

    -- What happened
    blocked BOOLEAN NOT NULL,
    masked_value TEXT,          -- Partially redacted value for debugging

    -- Request context
    request_path TEXT,
    request_method VARCHAR(10),
    request_ip INET,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pii_violations_timestamp ON privacy.pii_violations(timestamp);
CREATE INDEX idx_pii_violations_field ON privacy.pii_violations(field);
CREATE INDEX idx_pii_violations_actor ON privacy.pii_violations(actor_id);

-- Prevent modifications to violation log (audit trail)
CREATE OR REPLACE FUNCTION privacy.prevent_violation_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'PII violation log entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER pii_violations_no_update
    BEFORE UPDATE ON privacy.pii_violations
    FOR EACH ROW
    EXECUTE FUNCTION privacy.prevent_violation_modification();

CREATE TRIGGER pii_violations_no_delete
    BEFORE DELETE ON privacy.pii_violations
    FOR EACH ROW
    EXECUTE FUNCTION privacy.prevent_violation_modification();

-----------------------------------------------------------
-- PRIVACY AUDIT ACTIONS
-----------------------------------------------------------

-- Add privacy-related action types to audit system
COMMENT ON TABLE privacy.depseudonymization_requests IS
'Tracks all requests to reveal real identity from pseudonym.
Audit actions: privacy.depseudo_requested, privacy.depseudo_approved, privacy.depseudo_rejected';

COMMENT ON TABLE privacy.depseudonymization_tokens IS
'Time-limited tokens for approved identity access.
Audit actions: privacy.depseudo_used';

COMMENT ON TABLE privacy.pii_violations IS
'Records all detected PII leak attempts.
Audit actions: privacy.pii_blocked';
