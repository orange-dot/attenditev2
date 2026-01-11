-- Initial database schemas for Serbia Government Interoperability Platform
-- Each module owns its own schema

-- Create schemas
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS cases;
CREATE SCHEMA IF NOT EXISTS documents;
CREATE SCHEMA IF NOT EXISTS audit;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable PostGIS for location data (if available)
-- CREATE EXTENSION IF NOT EXISTS postgis;

-----------------------------------------------------------
-- IDENTITY SCHEMA
-----------------------------------------------------------

-- Agencies table
CREATE TABLE identity.agencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    parent_id UUID REFERENCES identity.agencies(id),

    -- Address
    address_street VARCHAR(255),
    address_city VARCHAR(100),
    address_postal_code VARCHAR(20),
    address_country VARCHAR(2) DEFAULT 'RS',
    address_lat DOUBLE PRECISION,
    address_lng DOUBLE PRECISION,

    -- Contact
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_mobile VARCHAR(50),

    -- Federation
    federation_cert BYTEA,

    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agencies_code ON identity.agencies(code);
CREATE INDEX idx_agencies_type ON identity.agencies(type);
CREATE INDEX idx_agencies_parent ON identity.agencies(parent_id);
CREATE INDEX idx_agencies_status ON identity.agencies(status);

-- Workers table
CREATE TABLE identity.workers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agency_id UUID NOT NULL REFERENCES identity.agencies(id),
    citizen_id UUID, -- Optional link to citizen record
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

CREATE INDEX idx_workers_agency ON identity.workers(agency_id);
CREATE INDEX idx_workers_email ON identity.workers(email);
CREATE INDEX idx_workers_status ON identity.workers(status);

-- Worker roles table (many-to-many)
CREATE TABLE identity.worker_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    worker_id UUID NOT NULL REFERENCES identity.workers(id) ON DELETE CASCADE,
    role VARCHAR(100) NOT NULL,
    scope VARCHAR(100) DEFAULT 'all',
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID REFERENCES identity.workers(id),

    UNIQUE(worker_id, role, scope)
);

CREATE INDEX idx_worker_roles_worker ON identity.worker_roles(worker_id);
CREATE INDEX idx_worker_roles_role ON identity.worker_roles(role);

-- Citizens table
CREATE TABLE identity.citizens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    jmbg VARCHAR(13) UNIQUE NOT NULL,

    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    date_of_birth DATE NOT NULL,
    gender VARCHAR(10),

    -- Address
    address_street VARCHAR(255),
    address_city VARCHAR(100),
    address_postal_code VARCHAR(20),
    address_country VARCHAR(2) DEFAULT 'RS',

    -- Contact
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_mobile VARCHAR(50),

    -- eID
    eid_linked BOOLEAN DEFAULT FALSE,
    eid_assurance VARCHAR(20), -- basic, high, highest

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_citizens_jmbg ON identity.citizens(jmbg);
CREATE INDEX idx_citizens_name ON identity.citizens(last_name, first_name);

-----------------------------------------------------------
-- CASES SCHEMA
-----------------------------------------------------------

-- Cases table (aggregate root)
CREATE TABLE cases.cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_number VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',

    title VARCHAR(255) NOT NULL,
    description TEXT,

    owning_agency_id UUID NOT NULL,
    lead_worker_id UUID NOT NULL,

    -- SLA
    sla_deadline TIMESTAMPTZ,
    sla_status VARCHAR(20) DEFAULT 'on_track',

    -- Sharing
    shared_with UUID[] DEFAULT '{}',
    access_levels JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ
);

CREATE INDEX idx_cases_number ON cases.cases(case_number);
CREATE INDEX idx_cases_type ON cases.cases(type);
CREATE INDEX idx_cases_status ON cases.cases(status);
CREATE INDEX idx_cases_owning_agency ON cases.cases(owning_agency_id);
CREATE INDEX idx_cases_lead_worker ON cases.cases(lead_worker_id);
CREATE INDEX idx_cases_sla_deadline ON cases.cases(sla_deadline);
CREATE INDEX idx_cases_shared_with ON cases.cases USING GIN(shared_with);

-- Case participants
CREATE TABLE cases.participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases.cases(id) ON DELETE CASCADE,
    citizen_id UUID,

    role VARCHAR(50) NOT NULL, -- applicant, subject, guardian, etc.
    name VARCHAR(255) NOT NULL,

    -- Contact (denormalized for convenience)
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),

    notes TEXT,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID NOT NULL
);

CREATE INDEX idx_participants_case ON cases.participants(case_id);
CREATE INDEX idx_participants_citizen ON cases.participants(citizen_id);

-- Case assignments
CREATE TABLE cases.assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases.cases(id) ON DELETE CASCADE,
    agency_id UUID NOT NULL,
    worker_id UUID NOT NULL,

    role VARCHAR(50) NOT NULL, -- lead, support, reviewer, observer
    status VARCHAR(50) NOT NULL DEFAULT 'active',

    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID NOT NULL,
    completed_at TIMESTAMPTZ,
    notes TEXT
);

CREATE INDEX idx_assignments_case ON cases.assignments(case_id);
CREATE INDEX idx_assignments_worker ON cases.assignments(worker_id);
CREATE INDEX idx_assignments_status ON cases.assignments(status);

-- Case events (timeline)
CREATE TABLE cases.case_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    case_id UUID NOT NULL REFERENCES cases.cases(id) ON DELETE CASCADE,

    type VARCHAR(50) NOT NULL,
    actor_id UUID NOT NULL,
    actor_agency_id UUID NOT NULL,

    description VARCHAR(500),
    data JSONB,

    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_case_events_case ON cases.case_events(case_id);
CREATE INDEX idx_case_events_type ON cases.case_events(type);
CREATE INDEX idx_case_events_timestamp ON cases.case_events(timestamp);

-----------------------------------------------------------
-- DOCUMENTS SCHEMA
-----------------------------------------------------------

-- Documents table
CREATE TABLE documents.documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_number VARCHAR(50) UNIQUE NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',

    title VARCHAR(255) NOT NULL,
    description TEXT,

    owner_agency_id UUID NOT NULL,
    created_by UUID NOT NULL,

    case_id UUID, -- Optional link to case

    current_version INT NOT NULL DEFAULT 1,

    -- Sharing
    shared_with UUID[] DEFAULT '{}',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_number ON documents.documents(document_number);
CREATE INDEX idx_documents_type ON documents.documents(type);
CREATE INDEX idx_documents_status ON documents.documents(status);
CREATE INDEX idx_documents_case ON documents.documents(case_id);
CREATE INDEX idx_documents_owner ON documents.documents(owner_agency_id);

-- Document versions
CREATE TABLE documents.versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents.documents(id) ON DELETE CASCADE,
    version INT NOT NULL,

    file_path VARCHAR(500) NOT NULL, -- MinIO path
    file_hash VARCHAR(64) NOT NULL, -- SHA-256
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NOT NULL,
    change_summary VARCHAR(500),

    UNIQUE(document_id, version)
);

CREATE INDEX idx_versions_document ON documents.versions(document_id);

-- Document signatures
CREATE TABLE documents.signatures (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents.documents(id) ON DELETE CASCADE,
    version INT NOT NULL,

    signer_id UUID NOT NULL,
    signer_agency_id UUID NOT NULL,

    type VARCHAR(20) NOT NULL, -- simple, advanced, qualified
    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    signature_data BYTEA,
    certificate BYTEA,
    timestamp_token BYTEA,

    reason VARCHAR(255),
    location VARCHAR(100),

    signed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_signatures_document ON documents.signatures(document_id);
CREATE INDEX idx_signatures_signer ON documents.signatures(signer_id);
CREATE INDEX idx_signatures_status ON documents.signatures(status);

-----------------------------------------------------------
-- AUDIT SCHEMA
-----------------------------------------------------------

-- Audit entries (append-only)
CREATE TABLE audit.entries (
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

-- Append-only: no UPDATE or DELETE allowed
CREATE INDEX idx_audit_timestamp ON audit.entries(timestamp);
CREATE INDEX idx_audit_actor ON audit.entries(actor_id);
CREATE INDEX idx_audit_action ON audit.entries(action);
CREATE INDEX idx_audit_resource ON audit.entries(resource_type, resource_id);
CREATE INDEX idx_audit_correlation ON audit.entries(correlation_id);

-- Prevent modifications to audit table
CREATE OR REPLACE FUNCTION audit.prevent_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_entries_no_update
    BEFORE UPDATE ON audit.entries
    FOR EACH ROW
    EXECUTE FUNCTION audit.prevent_modification();

CREATE TRIGGER audit_entries_no_delete
    BEFORE DELETE ON audit.entries
    FOR EACH ROW
    EXECUTE FUNCTION audit.prevent_modification();

-----------------------------------------------------------
-- HELPER FUNCTIONS
-----------------------------------------------------------

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to all tables with updated_at
CREATE TRIGGER update_agencies_timestamp
    BEFORE UPDATE ON identity.agencies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_workers_timestamp
    BEFORE UPDATE ON identity.workers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_citizens_timestamp
    BEFORE UPDATE ON identity.citizens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_cases_timestamp
    BEFORE UPDATE ON cases.cases
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_documents_timestamp
    BEFORE UPDATE ON documents.documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
