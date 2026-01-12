-- Federation schema for cross-agency trust and communication
-- Migration: 003_federation_schema.sql

-----------------------------------------------------------
-- FEDERATION SCHEMA
-----------------------------------------------------------

CREATE SCHEMA IF NOT EXISTS federation;

-- Trusted agencies registered with the Trust Authority
CREATE TABLE federation.trusted_agencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    gateway_url VARCHAR(500) NOT NULL,

    -- Cryptographic identity
    public_key BYTEA NOT NULL,
    certificate BYTEA NOT NULL,

    -- Trust status
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, suspended, revoked

    -- Timestamps
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trusted_agencies_code ON federation.trusted_agencies(code);
CREATE INDEX idx_trusted_agencies_status ON federation.trusted_agencies(status);

-- Service endpoints offered by agencies
CREATE TABLE federation.service_endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agency_id UUID NOT NULL REFERENCES federation.trusted_agencies(id) ON DELETE CASCADE,

    service_type VARCHAR(100) NOT NULL, -- e.g., "case.share", "document.verify"
    path VARCHAR(500) NOT NULL,
    version VARCHAR(20) NOT NULL,

    active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(agency_id, service_type, version)
);

CREATE INDEX idx_service_endpoints_agency ON federation.service_endpoints(agency_id);
CREATE INDEX idx_service_endpoints_type ON federation.service_endpoints(service_type);
CREATE INDEX idx_service_endpoints_active ON federation.service_endpoints(active);

-- Federation request log (for audit purposes)
CREATE TABLE federation.request_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Request details
    request_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Parties
    source_agency VARCHAR(50) NOT NULL,
    target_agency VARCHAR(50) NOT NULL,

    -- Request
    method VARCHAR(10) NOT NULL,
    path VARCHAR(500) NOT NULL,

    -- Response
    status_code INT,
    response_time_ms INT,

    -- Verification
    signature_valid BOOLEAN,
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_request_log_timestamp ON federation.request_log(timestamp);
CREATE INDEX idx_request_log_source ON federation.request_log(source_agency);
CREATE INDEX idx_request_log_target ON federation.request_log(target_agency);
CREATE INDEX idx_request_log_request_id ON federation.request_log(request_id);

-- Prevent modifications to request log (audit trail)
CREATE OR REPLACE FUNCTION federation.prevent_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Federation request log entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER request_log_no_update
    BEFORE UPDATE ON federation.request_log
    FOR EACH ROW
    EXECUTE FUNCTION federation.prevent_log_modification();

CREATE TRIGGER request_log_no_delete
    BEFORE DELETE ON federation.request_log
    FOR EACH ROW
    EXECUTE FUNCTION federation.prevent_log_modification();

-----------------------------------------------------------
-- COMMENTS
-----------------------------------------------------------

COMMENT ON TABLE federation.trusted_agencies IS
'Registry of trusted agencies that can participate in cross-agency communication.
Each agency has a public key for signature verification.';

COMMENT ON TABLE federation.service_endpoints IS
'Service endpoints exposed by agencies for federation.
Allows service discovery across the platform.';

COMMENT ON TABLE federation.request_log IS
'Immutable log of all federation requests for audit purposes.';
