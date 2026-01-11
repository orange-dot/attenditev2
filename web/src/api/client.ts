import type {
  Agency,
  Worker,
  Case,
  CaseEvent,
  Document,
  DocumentVersion,
  AuditEntry,
  AuditVerifyResponse,
  FederatedAgency,
  AnalysisResponse,
  AIExample,
} from '../types'

const API_BASE = '/api/v1'

class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public details?: Record<string, string>
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE}${endpoint}`

  const config: RequestInit = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  }

  const response = await fetch(url, config)

  if (!response.ok) {
    const error = await response.json().catch(() => ({}))
    throw new ApiError(
      response.status,
      error.code || 'UNKNOWN_ERROR',
      error.error || error.message || 'An error occurred',
      error.details
    )
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return {} as T
  }

  return response.json()
}

// Health checks are at root level, not under /api/v1
async function healthRequest<T>(endpoint: string): Promise<T> {
  const response = await fetch(endpoint)
  if (!response.ok) {
    throw new ApiError(response.status, 'HEALTH_ERROR', 'Health check failed')
  }
  return response.json()
}

export const api = {
  // Health (at root level, not /api/v1)
  health: () => healthRequest<{ status: string }>('/health'),
  ready: () => healthRequest<{ status: string; checks: Record<string, string> }>('/ready'),

  // Agencies
  agencies: {
    list: (params?: Record<string, string>) => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return request<{ data: Agency[]; total: number }>(`/agencies${query}`)
    },
    get: (id: string) => request<Agency>(`/agencies/${id}`),
    create: (data: unknown) => request<Agency>('/agencies', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) => request<Agency>(`/agencies/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: string) => request(`/agencies/${id}`, { method: 'DELETE' }),
    workers: (id: string) => request<{ data: Worker[]; total: number }>(`/agencies/${id}/workers`),
    createWorker: (agencyId: string, data: unknown) =>
      request<Worker>(`/agencies/${agencyId}/workers`, { method: 'POST', body: JSON.stringify(data) }),
  },

  // Cases
  cases: {
    list: (params?: Record<string, string>) => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return request<{ data: Case[]; total: number }>(`/cases${query}`)
    },
    get: (id: string) => request<Case>(`/cases/${id}`),
    create: (data: unknown) => request<Case>('/cases', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) => request<Case>(`/cases/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    open: (id: string) => request<Case>(`/cases/${id}/open`, { method: 'POST' }),
    start: (id: string) => request<Case>(`/cases/${id}/start`, { method: 'POST' }),
    close: (id: string, resolution: string) =>
      request<Case>(`/cases/${id}/close`, { method: 'POST', body: JSON.stringify({ resolution }) }),
    share: (id: string, data: unknown) => request(`/cases/${id}/share`, { method: 'POST', body: JSON.stringify(data) }),
    events: (id: string) => request<{ data: CaseEvent[]; total: number }>(`/cases/${id}/events`),
  },

  // Documents
  documents: {
    list: (params?: Record<string, string>) => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return request<{ data: Document[]; total: number }>(`/documents${query}`)
    },
    get: (id: string) => request<Document>(`/documents/${id}`),
    create: (data: unknown) => request<Document>('/documents', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: unknown) =>
      request<Document>(`/documents/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    archive: (id: string) => request<Document>(`/documents/${id}/archive`, { method: 'POST' }),
    versions: (id: string) => request<{ data: DocumentVersion[]; total: number }>(`/documents/${id}/versions`),
    signatures: (id: string) => request<{ data: unknown[]; total: number }>(`/documents/${id}/signatures`),
    requestSignature: (id: string, data: unknown) =>
      request(`/documents/${id}/signatures`, { method: 'POST', body: JSON.stringify(data) }),
    sign: (docId: string, sigId: string) =>
      request(`/documents/${docId}/signatures/${sigId}/sign`, { method: 'POST' }),
  },

  // Audit
  audit: {
    list: (params?: Record<string, string>) => {
      const query = params ? '?' + new URLSearchParams(params).toString() : ''
      return request<{ data: AuditEntry[]; total: number }>(`/audit${query}`)
    },
    get: (id: string) => request<AuditEntry>(`/audit/${id}`),
    verify: (details?: boolean) => request<AuditVerifyResponse>(`/audit/verify${details ? '?details=true' : ''}`),
    byResource: (type: string, id: string) => request<{ data: AuditEntry[] }>(`/audit/resource/${type}/${id}`),
  },

  // Federation
  federation: {
    agencies: () => request<{ data: FederatedAgency[]; total: number }>('/federation/trust/agencies'),
    services: (agencyId: string) => request<{ data: unknown[] }>(`/federation/trust/agencies/${agencyId}/services`),
  },

  // AI
  ai: {
    health: () => request<{ status: string }>('/ai/health'),
    analyze: (data: { document_text: string; document_type?: string }) =>
      request<AnalysisResponse>('/ai/analyze', { method: 'POST', body: JSON.stringify(data) }),
    examples: () => request<{ examples: AIExample[] }>('/ai/examples'),
  },

  // Simulation
  simulation: {
    start: (data: { use_case_id: string; use_case_title: string; total_steps: number; citizen_jmbg?: string }) =>
      request<{ success: boolean; session_id: string; message: string; timestamp: string }>(
        '/simulation/start',
        { method: 'POST', body: JSON.stringify(data) }
      ),
    step: (data: {
      use_case_id: string;
      use_case_title: string;
      session_id: string;
      citizen_jmbg?: string;
      step: {
        step_id: string;
        from_institution: string;
        to_institution: string;
        action: string;
        description: string;
        data_exchanged: string[];
        is_response: boolean;
      };
    }) =>
      request<{ success: boolean; audit_entry_id: string; timestamp: string; message: string }>(
        '/simulation/step',
        { method: 'POST', body: JSON.stringify(data) }
      ),
    complete: (data: { session_id: string; use_case_id: string; use_case_title: string; total_steps: number; success: boolean }) =>
      request<{ success: boolean; message: string; timestamp: string }>(
        '/simulation/complete',
        { method: 'POST', body: JSON.stringify(data) }
      ),
    institutions: () =>
      request<{ data: { id: string; name: string; type: string; city: string }[]; total: number }>('/simulation/institutions'),
  },
}

export { ApiError }
