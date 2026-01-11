-- Platform Database Initialization
-- This script runs automatically on first container start

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search

-- Set timezone
SET timezone = 'Europe/Belgrade';

-- Create schemas
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS cases;
CREATE SCHEMA IF NOT EXISTS documents;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS federation;
CREATE SCHEMA IF NOT EXISTS coordination;

-- Grant permissions
GRANT ALL ON SCHEMA identity TO platform;
GRANT ALL ON SCHEMA cases TO platform;
GRANT ALL ON SCHEMA documents TO platform;
GRANT ALL ON SCHEMA audit TO platform;
GRANT ALL ON SCHEMA federation TO platform;
GRANT ALL ON SCHEMA coordination TO platform;

-----------------------------------------------------------
-- IDENTITY SCHEMA (for seed data)
-----------------------------------------------------------

-- Agencies table
CREATE TABLE IF NOT EXISTS identity.agencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    parent_id UUID REFERENCES identity.agencies(id),
    address_street VARCHAR(255),
    address_city VARCHAR(100),
    address_postal_code VARCHAR(20),
    address_country VARCHAR(2) DEFAULT 'RS',
    address_lat DOUBLE PRECISION,
    address_lng DOUBLE PRECISION,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_mobile VARCHAR(50),
    federation_cert BYTEA,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agencies_code ON identity.agencies(code);
CREATE INDEX IF NOT EXISTS idx_agencies_type ON identity.agencies(type);
CREATE INDEX IF NOT EXISTS idx_agencies_parent ON identity.agencies(parent_id);

-- Workers table
CREATE TABLE IF NOT EXISTS identity.workers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agency_id UUID NOT NULL REFERENCES identity.agencies(id),
    citizen_id UUID,
    employee_id VARCHAR(100) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    position VARCHAR(100),
    department VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(agency_id, email),
    UNIQUE(agency_id, employee_id)
);

CREATE INDEX IF NOT EXISTS idx_workers_agency ON identity.workers(agency_id);
CREATE INDEX IF NOT EXISTS idx_workers_email ON identity.workers(email);

-----------------------------------------------------------
-- CASES SCHEMA (for seed data)
-----------------------------------------------------------

-- Cases table
CREATE TABLE IF NOT EXISTS cases.cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_number VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    title VARCHAR(255) NOT NULL,
    description TEXT,
    owning_agency_id UUID NOT NULL,
    lead_worker_id UUID NOT NULL,
    sla_deadline TIMESTAMPTZ,
    sla_status VARCHAR(20) DEFAULT 'on_track',
    shared_with UUID[] DEFAULT '{}',
    access_levels JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_cases_number ON cases.cases(case_number);
CREATE INDEX IF NOT EXISTS idx_cases_status ON cases.cases(status);
CREATE INDEX IF NOT EXISTS idx_cases_owning_agency ON cases.cases(owning_agency_id);

-- Case events
CREATE TABLE IF NOT EXISTS cases.case_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases.cases(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    actor_agency_id UUID NOT NULL,
    description VARCHAR(500),
    data JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_case_events_case ON cases.case_events(case_id);

-----------------------------------------------------------
-- DOCUMENTS SCHEMA (for seed data)
-----------------------------------------------------------

-- Documents table
CREATE TABLE IF NOT EXISTS documents.documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_number VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    title VARCHAR(255) NOT NULL,
    description TEXT,
    owner_agency_id UUID NOT NULL,
    created_by UUID NOT NULL,
    case_id UUID,
    current_version INT NOT NULL DEFAULT 1,
    shared_with UUID[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_number ON documents.documents(document_number);
CREATE INDEX IF NOT EXISTS idx_documents_case ON documents.documents(case_id);

-- Create read-only user for reporting (optional)
-- CREATE USER platform_readonly WITH PASSWORD 'readonly_password';
-- GRANT CONNECT ON DATABASE platform TO platform_readonly;
-- GRANT USAGE ON SCHEMA cases, documents, audit TO platform_readonly;
-- GRANT SELECT ON ALL TABLES IN SCHEMA cases, documents, audit TO platform_readonly;

-- Audit entries table (append-only) - matching application migration schema
CREATE TABLE IF NOT EXISTS audit.entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sequence BIGSERIAL UNIQUE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Hash chain for tamper detection
    hash VARCHAR(64) NOT NULL,
    prev_hash VARCHAR(64),

    -- Actor
    actor_type VARCHAR(20) NOT NULL, -- citizen, worker, system, external
    actor_id UUID NOT NULL,
    actor_agency_id UUID,
    actor_ip VARCHAR(45),
    actor_device VARCHAR(255),

    -- Action
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,

    -- Changes
    changes JSONB,

    -- Context
    correlation_id UUID,
    session_id UUID,
    justification VARCHAR(500)
);

-- Create index for audit queries
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit.entries(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_sequence ON audit.entries(sequence DESC);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit.entries(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit.entries(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_correlation ON audit.entries(correlation_id);

-- Make audit table append-only (no updates/deletes)
CREATE OR REPLACE FUNCTION audit.prevent_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS audit_prevent_update ON audit.entries;
CREATE TRIGGER audit_prevent_update
    BEFORE UPDATE OR DELETE ON audit.entries
    FOR EACH ROW EXECUTE FUNCTION audit.prevent_modification();

-- Audit checkpoints table - external witness for tamper evidence
CREATE TABLE IF NOT EXISTS audit.checkpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    checkpoint_hash VARCHAR(64) NOT NULL,      -- Hash of the chain at this point
    last_sequence BIGINT NOT NULL,             -- Sequence number of last entry
    last_entry_id UUID NOT NULL,               -- ID of last entry
    entry_count INT NOT NULL,                  -- Total entries at checkpoint time

    -- Witness information
    witness_type VARCHAR(50) NOT NULL,         -- 'local', 'opentimestamps', 'rfc3161_tsa'
    witness_proof BYTEA,                       -- Proof from external service (OTS file, TSA response)
    witness_url TEXT,                          -- URL to verify (e.g., blockchain explorer)
    witness_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'confirmed', 'failed'

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ                   -- When external witness confirmed
);

CREATE INDEX IF NOT EXISTS idx_checkpoint_sequence ON audit.checkpoints(last_sequence DESC);
CREATE INDEX IF NOT EXISTS idx_checkpoint_created ON audit.checkpoints(created_at DESC);

-- Checkpoints are also append-only
CREATE TRIGGER checkpoint_prevent_update
    BEFORE UPDATE OR DELETE ON audit.checkpoints
    FOR EACH ROW EXECUTE FUNCTION audit.prevent_modification();

-- Coordination events table
CREATE TABLE IF NOT EXISTS coordination.events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    subject_jmbg VARCHAR(13) NOT NULL,
    subject_name VARCHAR(200),
    source_system VARCHAR(50) NOT NULL,
    source_agency VARCHAR(50) NOT NULL,
    source_reference VARCHAR(100),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    details JSONB,
    metadata JSONB,
    enrichment JSONB,
    target_agencies TEXT[],
    acknowledged JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for coordination
CREATE INDEX IF NOT EXISTS idx_coord_status ON coordination.events(status);
CREATE INDEX IF NOT EXISTS idx_coord_priority ON coordination.events(priority);
CREATE INDEX IF NOT EXISTS idx_coord_subject ON coordination.events(subject_jmbg);
CREATE INDEX IF NOT EXISTS idx_coord_created ON coordination.events(created_at DESC);

-- Notifications table
CREATE TABLE IF NOT EXISTS coordination.notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID REFERENCES coordination.events(id),
    notification_type VARCHAR(20) NOT NULL,
    priority VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    recipient_id VARCHAR(100) NOT NULL,
    recipient_type VARCHAR(20) NOT NULL,
    recipient_name VARCHAR(200),
    phone VARCHAR(50),
    email VARCHAR(200),
    device_token TEXT,
    subject VARCHAR(500),
    body TEXT,
    data JSONB,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    retry_count INT DEFAULT 0,
    last_retry_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for notifications
CREATE INDEX IF NOT EXISTS idx_notif_status ON coordination.notifications(status);
CREATE INDEX IF NOT EXISTS idx_notif_recipient ON coordination.notifications(recipient_id);
CREATE INDEX IF NOT EXISTS idx_notif_scheduled ON coordination.notifications(scheduled_at);

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully';
END $$;
