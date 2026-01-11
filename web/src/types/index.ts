// Agency types
export interface Agency {
  id: string
  code?: string
  name: string
  type: string
  status: string
  jurisdiction?: string
  created_at: string
  updated_at: string
}

export interface CreateAgencyRequest {
  name: string
  type: string
  jurisdiction: string
}

export interface Worker {
  id: string
  agency_id: string
  email: string
  name: string
  first_name?: string
  last_name?: string
  role: string
  status: string
  created_at: string
}

export interface CreateWorkerRequest {
  name: string
  email: string
  role: string
}

// Case types
export interface Case {
  id: string
  case_number?: string
  type: string
  status: string
  priority: string
  title: string
  description?: string
  owner_agency_id?: string
  lead_agency_id?: string
  created_at: string
  updated_at: string
}

export interface CreateCaseRequest {
  title: string
  description?: string
  type: string
  priority: string
  lead_agency_id?: string
}

export interface CaseEvent {
  id: string
  case_id: string
  event_type: string
  description?: string
  timestamp: string
  created_at?: string
  actor_id?: string
  metadata?: Record<string, unknown>
}

// Document types
export interface Document {
  id: string
  document_number?: string
  type: string
  status: string
  title: string
  description?: string
  content?: string
  version: number
  owner_agency_id?: string
  case_id?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateDocumentRequest {
  title: string
  type: string
  content: string
  case_id?: string
  metadata?: Record<string, unknown>
}

export interface DocumentVersion {
  version: number
  hash?: string
  created_at: string
  created_by?: string
}

export interface SignatureRequest {
  signer_id: string
  reason?: string
}

// Audit types
export interface AuditEntry {
  id: string
  actor_id?: string
  actor_type?: string
  action: string
  resource_type: string
  resource_id: string
  metadata?: Record<string, unknown>
  hash?: string
  timestamp: string
  created_at?: string
}

export interface AuditVerifyEntry {
  id: string
  sequence: number
  hash: string
  prev_hash: string
  valid: boolean
  action: string
}

export interface AuditVerifyResponse {
  valid: boolean
  checked: number
  violations?: string[]
  entries?: AuditVerifyEntry[]
}

// AI types
export interface AnalysisRequest {
  document_text: string
  document_type?: string
  patient_context?: Record<string, unknown>
}

export interface Anomaly {
  type: string
  severity: string
  title: string
  description: string
  evidence: string[]
  recommendation: string
  protocol_reference?: string
}

export interface AnalysisResponse {
  request_id: string
  timestamp: string
  anomalies_found: number
  anomalies: Anomaly[]
  processing_time_ms: number
  model_used: string
  confidence: number
}

export interface AIExample {
  id: string
  title: string
  description: string
  document_text: string
  expected_anomaly?: string
}

// Federation types
export interface FederatedAgency {
  id: string
  name: string
  code: string
  gateway_url: string
  status: string
  registered_at: string
  last_seen_at: string
}

// API response types
export interface ListResponse<T> {
  data: T[]
  total: number
}

export interface HealthStatus {
  status: string
  checks?: Record<string, string>
}
